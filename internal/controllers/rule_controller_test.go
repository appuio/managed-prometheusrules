package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	testWatchedNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "watched",
		},
	}
	testWatchedRule = &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rule-01",
			Namespace: testWatchedNamespace.GetName(),
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{{
				Name:  "group-name",
				Rules: []monitoringv1.Rule{{Alert: "alert-name"}},
			}},
		},
	}
	testOtherNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "other",
		},
	}
	testOtherRule = &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rule-02",
			Namespace: testOtherNamespace.GetName(),
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{{
				Name:  "group-name",
				Rules: []monitoringv1.Rule{{Alert: "alert-name"}},
			}},
		},
	}
	testManagedNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "managed",
		},
	}
	testManagedRule = &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "watched-test-rule-01",
			Namespace: testManagedNamespace.GetName(),
			Annotations: map[string]string{
				managedRuleOwnerName:      testWatchedRule.GetName(),
				managedRuleOwnerNamespace: testWatchedRule.GetNamespace(),
			},
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{{
				Name: "group-name",
				Rules: []monitoringv1.Rule{{
					Alert: "alert-name",
					Labels: map[string]string{
						"syn": "true",
					},
				}},
			}},
		},
	}

	typeWatchedName = types.NamespacedName{
		Name:      testWatchedRule.GetName(),
		Namespace: testWatchedRule.GetNamespace(),
	}
	typeOtherName = types.NamespacedName{
		Name:      testOtherRule.GetName(),
		Namespace: testOtherRule.GetNamespace(),
	}
	typeManagedName = types.NamespacedName{
		Name:      testManagedRule.GetName(),
		Namespace: testManagedRule.GetNamespace(),
	}
)

var _ = Describe("Rule Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		BeforeEach(func() {
			By("creating the custom resources for the controller")
			mns := testManagedNamespace.DeepCopy()
			err := k8sClient.Get(ctx, types.NamespacedName{Name: mns.GetName()}, mns)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, mns)).To(Succeed())
			}
			wns := testWatchedNamespace.DeepCopy()
			err = k8sClient.Get(ctx, types.NamespacedName{Name: wns.GetName()}, wns)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, wns)).To(Succeed())
			}
			wr := testWatchedRule.DeepCopy()
			err = k8sClient.Get(ctx, typeWatchedName, wr)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, wr)).To(Succeed())
			}
			ons := testOtherNamespace.DeepCopy()
			err = k8sClient.Get(ctx, types.NamespacedName{Name: ons.GetName()}, ons)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, ons)).To(Succeed())
			}
			or := testOtherRule.DeepCopy()
			err = k8sClient.Get(ctx, typeOtherName, or)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, or)).To(Succeed())
			}
		})

		AfterEach(func() {
			By("removing the custom resources for the controller")
			mr := testManagedRule.DeepCopy()
			err := k8sClient.Get(ctx, typeManagedName, mr)
			if err == nil {
				Expect(k8sClient.Delete(ctx, mr)).To(Succeed())
			}
		})

		It("should successfully create a managed resource", func() {
			By("Reconciling the watched resource")
			controllerReconciler := &RuleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),

				ManagedNamespace:  testManagedNamespace.GetName(),
				WatchedNamespaces: []string{},
				WatchedRegex:      testWatchedNamespace.GetName(),
				DryRun:            false,
				ExternalParser:    "",
				ExternalParams:    "",
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeWatchedName,
			})
			Expect(err).NotTo(HaveOccurred())

			mr := &monitoringv1.PrometheusRule{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      testManagedRule.GetName(),
				Namespace: testManagedNamespace.GetName(),
			}, mr)
			Expect(err).NotTo(HaveOccurred())
			Expect(mr.GetName()).To(Equal(testManagedRule.GetName()))
			Expect(mr.GetNamespace()).To(Equal(testManagedRule.GetNamespace()))
			Expect(mr.GetAnnotations()).To(Equal(testManagedRule.GetAnnotations()))
			Expect(mr.Spec).To(Equal(testManagedRule.Spec))
		})

		It("should not create a managed resource, when dry-run enabled", func() {
			By("Reconciling the watched resource")
			controllerReconciler := &RuleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),

				ManagedNamespace:  testManagedNamespace.GetName(),
				WatchedNamespaces: []string{},
				WatchedRegex:      testWatchedNamespace.GetName(),
				DryRun:            true,
				ExternalParser:    "",
				ExternalParams:    "",
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeWatchedName,
			})
			Expect(err).NotTo(HaveOccurred())

			list := &monitoringv1.PrometheusRuleList{}
			err = k8sClient.List(ctx, list, &client.ListOptions{
				Namespace: testManagedRule.GetNamespace(),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(list.Items)).To(Equal(0))
		})

		It("should successfully delete the managed rule when watched rule is deleted", func() {
			By("Create the managed resource")
			mr := testManagedRule.DeepCopy()
			Expect(k8sClient.Create(ctx, mr)).To(Succeed())

			By("Deleting the watched resource")
			wr := testWatchedRule.DeepCopy()
			Expect(k8sClient.Delete(ctx, wr)).To(Succeed())

			By("Reconciling the managed resource")
			controllerReconciler := &RuleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),

				ManagedNamespace:  testManagedNamespace.GetName(),
				WatchedNamespaces: []string{},
				WatchedRegex:      testWatchedNamespace.GetName(),
				DryRun:            false,
				ExternalParser:    "",
				ExternalParams:    "",
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeManagedName,
			})
			Expect(err).NotTo(HaveOccurred())

			list := &monitoringv1.PrometheusRuleList{}
			err = k8sClient.List(ctx, list, &client.ListOptions{
				Namespace: testManagedRule.GetNamespace(),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(list.Items)).To(Equal(0))
		})

		It("should successfully delete the managed rule when namespace is not watched", func() {
			By("Create the managed resource")
			mr := testManagedRule.DeepCopy()
			Expect(k8sClient.Create(ctx, mr)).To(Succeed())

			By("Reconciling the managed resource")
			controllerReconciler := &RuleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),

				ManagedNamespace:  testManagedNamespace.GetName(),
				WatchedNamespaces: []string{},
				WatchedRegex:      testOtherNamespace.GetName(),
				DryRun:            false,
				ExternalParser:    "",
				ExternalParams:    "",
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeManagedName,
			})
			Expect(err).NotTo(HaveOccurred())

			list := &monitoringv1.PrometheusRuleList{}
			err = k8sClient.List(ctx, list, &client.ListOptions{
				Namespace: testManagedRule.GetNamespace(),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(list.Items)).To(Equal(0))
		})

		It("should successfully delete the managed rule when annotations not set", func() {
			By("Updating the managed resource")
			mr := testManagedRule.DeepCopy()
			mr.SetAnnotations(map[string]string{})
			Expect(k8sClient.Create(ctx, mr)).To(Succeed())

			By("Reconciling the managed resource")
			controllerReconciler := &RuleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),

				ManagedNamespace:  testManagedNamespace.GetName(),
				WatchedNamespaces: []string{},
				WatchedRegex:      testWatchedNamespace.GetName(),
				DryRun:            false,
				ExternalParser:    "",
				ExternalParams:    "",
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeManagedName,
			})
			Expect(err).NotTo(HaveOccurred())

			list := &monitoringv1.PrometheusRuleList{}
			err = k8sClient.List(ctx, list, &client.ListOptions{
				Namespace: testManagedRule.GetNamespace(),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(list.Items)).To(Equal(0))
		})

		It("should not reconcile rule when namespace is not watched", func() {
			By("Reconciling the other resource")
			controllerReconciler := &RuleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),

				ManagedNamespace:  testManagedNamespace.GetName(),
				WatchedNamespaces: []string{},
				WatchedRegex:      testWatchedNamespace.GetName(),
				// WatchedNamespaces: []string{testWatchedNamespace.GetName()},
				// WatchedRegex:      "",
				DryRun:         false,
				ExternalParser: "",
				ExternalParams: "",
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeOtherName,
			})
			Expect(err).NotTo(HaveOccurred())

			list := &monitoringv1.PrometheusRuleList{}
			err = k8sClient.List(ctx, list, &client.ListOptions{
				Namespace: testManagedRule.GetNamespace(),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(list.Items)).To(Equal(0))
		})

		It("should update the managed resource, when watched rule changes", func() {
			By("Create the managed resource")
			mr := testManagedRule.DeepCopy()
			Expect(k8sClient.Create(ctx, mr)).To(Succeed())

			By("Update the watched resource")
			wr := &monitoringv1.PrometheusRule{}
			Expect(k8sClient.Get(ctx, typeWatchedName, wr)).To(Succeed())
			wr.Spec.Groups[0].Name = "new-name"
			Expect(k8sClient.Update(ctx, wr)).To(Succeed())

			By("Reconciling the managed resource")
			controllerReconciler := &RuleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),

				ManagedNamespace:  testManagedNamespace.GetName(),
				WatchedNamespaces: []string{},
				WatchedRegex:      testWatchedNamespace.GetName(),
				DryRun:            false,
				ExternalParser:    "",
				ExternalParams:    "",
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeWatchedName,
			})
			Expect(err).NotTo(HaveOccurred())

			should := testManagedRule.DeepCopy()
			should.Spec.Groups[0].Name = "new-name"
			is := &monitoringv1.PrometheusRule{}
			err = k8sClient.Get(ctx, typeManagedName, is)
			Expect(err).NotTo(HaveOccurred())
			Expect(is.GetName()).To(Equal(should.GetName()))
			Expect(is.GetNamespace()).To(Equal(should.GetNamespace()))
			Expect(is.GetAnnotations()).To(Equal(should.GetAnnotations()))
			Expect(is.Spec).To(Equal(should.Spec))
		})

		It("should not update the managed resource, when watched rule changes and dry-run enabled", func() {
			By("Create the managed resource")
			mr := testManagedRule.DeepCopy()
			Expect(k8sClient.Create(ctx, mr)).To(Succeed())

			By("Update the watched resource")
			wr := &monitoringv1.PrometheusRule{}
			Expect(k8sClient.Get(ctx, typeWatchedName, wr)).To(Succeed())
			wr.Spec.Groups[0].Name = "new-name"
			Expect(k8sClient.Update(ctx, wr)).To(Succeed())

			By("Reconciling the watched resource")
			controllerReconciler := &RuleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),

				ManagedNamespace:  testManagedNamespace.GetName(),
				WatchedNamespaces: []string{},
				WatchedRegex:      testWatchedNamespace.GetName(),
				DryRun:            true,
				ExternalParser:    "",
				ExternalParams:    "",
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeWatchedName,
			})
			Expect(err).NotTo(HaveOccurred())

			should := testManagedRule.DeepCopy()
			is := &monitoringv1.PrometheusRule{}
			err = k8sClient.Get(ctx, typeManagedName, is)
			Expect(err).NotTo(HaveOccurred())
			Expect(is.GetName()).To(Equal(should.GetName()))
			Expect(is.GetNamespace()).To(Equal(should.GetNamespace()))
			Expect(is.GetAnnotations()).To(Equal(should.GetAnnotations()))
			Expect(is.Spec).To(Equal(should.Spec))
		})

		It("should update the managed resource, when reconciling managed resource", func() {
			By("Create the managed resource")
			mr := testManagedRule.DeepCopy()
			Expect(k8sClient.Create(ctx, mr)).To(Succeed())

			By("Update the watched resource")
			wr := &monitoringv1.PrometheusRule{}
			Expect(k8sClient.Get(ctx, typeWatchedName, wr)).To(Succeed())
			wr.Spec.Groups[0].Name = "new-name"
			Expect(k8sClient.Update(ctx, wr)).To(Succeed())

			By("Reconciling the managed resource")
			controllerReconciler := &RuleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),

				ManagedNamespace:  testManagedNamespace.GetName(),
				WatchedNamespaces: []string{},
				WatchedRegex:      testWatchedNamespace.GetName(),
				DryRun:            false,
				ExternalParser:    "",
				ExternalParams:    "",
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeManagedName,
			})
			Expect(err).NotTo(HaveOccurred())

			should := testManagedRule.DeepCopy()
			should.Spec.Groups[0].Name = "new-name"
			is := &monitoringv1.PrometheusRule{}
			err = k8sClient.Get(ctx, typeManagedName, is)
			Expect(err).NotTo(HaveOccurred())
			Expect(is.GetName()).To(Equal(should.GetName()))
			Expect(is.GetNamespace()).To(Equal(should.GetNamespace()))
			Expect(is.GetAnnotations()).To(Equal(should.GetAnnotations()))
			Expect(is.Spec).To(Equal(should.Spec))
		})

		It("should successfully abort if reconciled rule does not exist", func() {
			By("Reconciling the managed resource")
			controllerReconciler := &RuleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),

				ManagedNamespace:  testManagedNamespace.GetName(),
				WatchedNamespaces: []string{},
				WatchedRegex:      testOtherNamespace.GetName(),
				DryRun:            false,
				ExternalParser:    "",
				ExternalParams:    "",
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeManagedName,
			})
			Expect(err).NotTo(HaveOccurred())

			list := &monitoringv1.PrometheusRuleList{}
			err = k8sClient.List(ctx, list, &client.ListOptions{
				Namespace: testManagedRule.GetNamespace(),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(list.Items)).To(Equal(0))
		})
	})
})
