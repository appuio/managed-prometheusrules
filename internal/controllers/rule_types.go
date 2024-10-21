package controllers

import (
	"crypto/sha1"
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

const (
	managedRuleOwnerName      = "managedrules.appuio.io/upstream-rule-name"
	managedRuleOwnerNamespace = "managedrules.appuio.io/upstream-rule-namespace"
)

func hasOwnerAnnotations(r *monitoringv1.PrometheusRule) bool {
	_, hasOwnerName := r.GetAnnotations()[managedRuleOwnerName]
	_, hasOwnerNamespace := r.GetAnnotations()[managedRuleOwnerNamespace]

	return hasOwnerName && hasOwnerNamespace
}

func isOwndeBy(r, by *monitoringv1.PrometheusRule) bool {
	hasOwnerName := getOwnerName(r) == by.GetName()
	hasOwnerNamespace := getOwnerNamespace(r) == by.GetNamespace()

	return hasOwnerName && hasOwnerNamespace
}

func getOwnerName(r *monitoringv1.PrometheusRule) string {
	return r.GetAnnotations()[managedRuleOwnerName]
}
func getOwnerNamespace(r *monitoringv1.PrometheusRule) string {
	return r.GetAnnotations()[managedRuleOwnerNamespace]
}

func hashedName(r *monitoringv1.PrometheusRule) string {
	hasher := sha1.New()
	hasher.Write([]byte(r.GetName()))

	// return fmt.Sprintf("%s-%s", r.GetNamespace(), hex.EncodeToString(hasher.Sum(nil)))
	return fmt.Sprintf("%s-%s", r.GetNamespace(), r.GetName())
}
