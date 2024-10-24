package controllers

import (
	"context"
	_ "embed"
	"encoding/json"
	"os"
	"regexp"
	"slices"

	"github.com/google/go-jsonnet"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// RuleReconciler reconciles a PrometheusRule object
type RuleReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	ManagedNamespace  string
	WatchedNamespaces []string
	WatchedRegex      string
	DryRun            bool
	ExternalParser    string
	ExternalParams    string
}

const (
	ruleTypeOwned   = "owned"
	ruleTypeWatched = "watched"
	ruleTypeIgnore  = "ignored"
)

var (
	//go:embed default_parser.jsonnet
	defaultParser []byte
)

//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,namespace=system,verbs=get;list;watch;update;patch;delete

// SetupWithManager sets up the controller with the Manager.
func (r *RuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1.PrometheusRule{}).
		Complete(r)
}

// Reconcile reacts to all PrometheusRule changes.
// If resource is a managed PrometheusRule check if the upstream PrometheusRule still exists,
// else check if a managed PrometheusRule exists and update/create from upstream PrometheusRule.
func (r *RuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	rule := &monitoringv1.PrometheusRule{}
	if err := r.Get(ctx, req.NamespacedName, rule); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	switch r.ruleType(rule) {
	case ruleTypeOwned:
		l.Info("reconcile owned rule")
		return r.reconcileOwnedRule(ctx, rule)
	case ruleTypeWatched:
		l.Info("reconcile watched rule")
		return r.reconcileWatchedRule(ctx, rule)
	}

	return ctrl.Result{}, nil
}

func (r *RuleReconciler) reconcileOwnedRule(ctx context.Context, ownedRule *monitoringv1.PrometheusRule) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// Check if annotations for upstream PrometheusRule is present.
	if !hasOwnerAnnotations(ownedRule) {
		l.Info("managed rule has no owner, deleting manged rule")
		r.Delete(ctx, ownedRule)
		return ctrl.Result{}, nil
	}

	// Get watched PrometheusRule from annotations.
	watchedRule := &monitoringv1.PrometheusRule{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      getOwnerName(ownedRule),
		Namespace: getOwnerNamespace(ownedRule),
	}, watchedRule); err != nil {
		l.Info("watched rule not found, deleting owned rule")
		r.Delete(ctx, ownedRule)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Delete managed PrometheusRule if watched rule is ignored.
	if r.ruleType(watchedRule) == ruleTypeIgnore {
		l.Info("watched rule is not in watched namespaces, deleting owned rule")
		r.Delete(ctx, ownedRule)
		return ctrl.Result{}, nil
	}

	// Update owned rule from watched rule.
	return r.updateFromWatched(ctx, ownedRule, watchedRule)
}

func (r *RuleReconciler) reconcileWatchedRule(ctx context.Context, watchedRule *monitoringv1.PrometheusRule) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	ownedRule := &monitoringv1.PrometheusRule{}

	// Get list of owned rules.
	list := &monitoringv1.PrometheusRuleList{}
	listOps := &client.ListOptions{
		Namespace: r.ManagedNamespace,
	}
	err := r.List(ctx, list, listOps)
	if err != nil {
		l.Info("can't list owned rules")
		return ctrl.Result{}, err
	}

	// Check list for owned rule.
	found := false
	for _, item := range list.Items {
		if isOwndeBy(item, watchedRule) {
			ownedRule = item
			found = true
			break
		}
	}
	if !found {
		l.Info("no owned rule found, create owned rule")
		return r.createNewFromWatched(ctx, watchedRule)
	}

	// Update owned rule from watched rule.
	l.Info("update owned rule")
	return r.updateFromWatched(ctx, ownedRule, watchedRule)
}

func (r *RuleReconciler) createNewFromWatched(ctx context.Context, watchedRule *monitoringv1.PrometheusRule) (ctrl.Result, error) {
	ownedRule, err := r.parseWatchedRule(watchedRule)
	if err != nil {
		return ctrl.Result{}, err
	}

	opts := &client.CreateOptions{}
	if r.DryRun {
		opts.DryRun = []string{"All"}
	}
	if err := r.Create(ctx, ownedRule, opts); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *RuleReconciler) updateFromWatched(ctx context.Context, ownedRule, watchedRule *monitoringv1.PrometheusRule) (ctrl.Result, error) {
	ownedRule, err := r.parseWatchedRuleWithMetadata(watchedRule, ownedRule.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}

	opts := &client.UpdateOptions{}
	if r.DryRun {
		opts.DryRun = []string{"All"}
	}
	if err := r.Update(ctx, ownedRule, opts); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RuleReconciler) parseWatchedRule(watchedRule *monitoringv1.PrometheusRule) (*monitoringv1.PrometheusRule, error) {

	return r.parseWatchedRuleWithMetadata(watchedRule, metav1.ObjectMeta{
		Name:      hashedName(watchedRule),
		Namespace: r.ManagedNamespace,
		Annotations: map[string]string{
			managedRuleOwnerName:      watchedRule.GetName(),
			managedRuleOwnerNamespace: watchedRule.GetNamespace(),
		},
	})
}
func (r *RuleReconciler) parseWatchedRuleWithMetadata(watchedRule *monitoringv1.PrometheusRule, meta metav1.ObjectMeta) (*monitoringv1.PrometheusRule, error) {
	// Create jsonnet VM and add external variables.
	vm := jsonnet.MakeVM()

	// Add watchedRule as external variable.
	watchedJson, err := json.Marshal(watchedRule.Spec)
	if err != nil {
		return nil, err
	}
	vm.ExtCode("rule", string(watchedJson))

	// Add external parameters as external variable.
	if r.ExternalParams != "" {
		params, err := os.ReadFile(r.ExternalParams)
		if err != nil {
			return nil, err
		}
		vm.ExtCode("params", string(params))
	}

	// Get Jsonnet snippet for parsing.
	parserName := "default_parser.jsonnet"
	parserSnippet := defaultParser
	if r.ExternalParser != "" {
		parserName = r.ExternalParser
		parserSnippet, err = os.ReadFile(r.ExternalParser)
		if err != nil {
			return nil, err
		}
	}

	// Run jsonnet parser.
	output, err := vm.EvaluateAnonymousSnippet(parserName, string(parserSnippet))
	if err != nil {
		return nil, err
	}

	specs := monitoringv1.PrometheusRuleSpec{}
	if err := json.Unmarshal([]byte(output), &specs); err != nil {
		return nil, err
	}

	// Return new rule from parsed specs.
	return &monitoringv1.PrometheusRule{
		ObjectMeta: meta,
		Spec:       specs,
	}, nil
}

func (r *RuleReconciler) ruleType(rule *monitoringv1.PrometheusRule) string {
	x := regexp.MustCompile(r.WatchedRegex)
	hasRegex := r.WatchedRegex != ""

	switch {
	case rule.GetNamespace() == r.ManagedNamespace:
		return ruleTypeOwned

	case slices.Contains(r.WatchedNamespaces, rule.GetNamespace()):
		return ruleTypeWatched

	case hasRegex && x.MatchString(rule.GetNamespace()):
		return ruleTypeWatched

	case r.WatchedRegex == "" && len(r.WatchedNamespaces) < 1:
		return ruleTypeWatched
	}

	return ruleTypeIgnore
}
