package v1

const (
	// ConditionTypeReady is used to signal whether a reconciliation has completed successfully.
	ConditionTypeReady = "Ready"

	// ConditionTypeKeyValid is set on an App when spec.validateKey is true and
	// reports the outcome of the signer's key validation. Absent when
	// spec.validateKey is false.
	ConditionTypeKeyValid = "KeyValid"

	// ConditionTypeExtraDataDegraded is an abnormal-true (kstatus-style)
	// condition: set True when spec.secret.extraData did not resolve in full
	// on the last reconcile — reason KeysMissing when optional listed keys
	// are absent, or reason SourceUnavailable while the managed Secret
	// serves last-known-good extraData because a required source cannot be
	// read. Absent when extraData resolves cleanly or none is configured.
	ConditionTypeExtraDataDegraded = "ExtraDataDegraded"

	// Condition reasons used by the Token and ClusterToken controllers when
	// resolving spec.appRef.

	// ReasonAppNotFound indicates the referenced App does not exist.
	ReasonAppNotFound = "AppNotFound"
	// ReasonAppNotReady indicates the referenced App exists but its Ready
	// condition is not True.
	ReasonAppNotReady = "AppNotReady"
	// ReasonNoStartupConfig indicates no spec.appRef was set and the operator
	// has no startup GitHub App configuration.
	ReasonNoStartupConfig = "NoStartupConfig"
	// ReasonReconciled indicates a successful reconciliation.
	ReasonReconciled = "Reconciled"
	// ReasonSetupFailed indicates construction of the GitHub App client failed.
	ReasonSetupFailed = "SetupFailed"
	// ReasonSecretNotFound indicates the Secret named by App.spec.keyRef
	// could not be fetched (typically NotFound).
	ReasonSecretNotFound = "SecretNotFound"
	// ReasonInvalidKey indicates the resolved key material is missing,
	// empty, or not a usable PEM-encoded RSA private key.
	ReasonInvalidKey = "InvalidKey"

	// Condition reasons used with ConditionTypeExtraDataDegraded (and, for
	// SourceUnavailable, with ConditionTypeReady when Secret creation is
	// blocked).

	// ReasonKeysMissing indicates optional extraData keys were absent from
	// their source and skipped.
	ReasonKeysMissing = "KeysMissing"
	// ReasonSourceUnavailable indicates a required extraData source could not
	// be resolved.
	ReasonSourceUnavailable = "SourceUnavailable"
)
