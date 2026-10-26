package apperror

func define(kind Kind, code, message string) Spec {
	return Spec{kind: kind, code: code, message: message}
}

// Common HTTP and validation failures.
var (
	InternalFailure   = define(Internal, "internal_error", "An unexpected error occurred.")
	InvalidRequest    = define(Invalid, "invalid_request", "Invalid request body.")
	RouteNotFound     = define(NotFound, "route_not_found", "Route not found.")
	UnsupportedMethod = define(MethodNotAllowed, "method_not_allowed", "Method not allowed.")
	CSRFFailed        = define(Forbidden, "csrf_failed", "Request origin could not be verified.")
	RateLimited       = define(TooManyRequests, "rate_limited", "Too many sign-in attempts. Please try again shortly.")
)

// Authentication and OAuth failures.
var (
	InvalidRegistration      = define(Invalid, "invalid_request", "A valid email and password (at least 6 characters, at most 72 UTF-8 bytes) are required.")
	EmailAlreadyRegistered   = define(Conflict, "email_already_registered", "Email is already registered.")
	InvalidLoginCredentials  = define(Unauthorized, "invalid_credentials", "Invalid email or password.")
	InvalidPassword          = define(Unauthorized, "invalid_credentials", "Invalid password.")
	AuthenticationRequired   = define(Unauthorized, "authentication_required", "Sign in to continue.")
	ReauthenticationRequired = define(Unauthorized, "reauthentication_required", "Sign in again or confirm your password before linking a login method.")
	UnsupportedProvider      = define(Invalid, "unsupported_provider", "Unsupported login provider.")
	ProviderUnavailable      = define(Unavailable, "provider_unavailable", "This login provider is not configured.")
	InvalidOAuthState        = define(Invalid, "invalid_oauth_state", "This sign-in request has expired. Please try again.")
	InvalidLinkSession       = define(Unauthorized, "invalid_link_session", "Sign in again before linking this account.")
	OAuthCancelled           = define(Invalid, "oauth_cancelled", "Social sign-in was cancelled. Please try again.")
	OAuthFailed              = define(Upstream, "oauth_failed", "Unable to verify your social account. Please try again.")
	AccountLinkRequired      = define(Conflict, "account_link_required", "This email already has an account. Sign in with your existing method, then link this provider in Settings.")
	IdentityAlreadyLinked    = define(Conflict, "identity_already_linked", "This login method is already linked to an account.")
)

// Wallet, payment, trustline and swap failures.
var (
	WalletNotFound            = define(NotFound, "wallet_not_found", "Wallet not found.")
	WalletNotReady            = define(Conflict, "wallet_not_ready", "Wallet is not ready yet.")
	UnsupportedChain          = define(Invalid, "unsupported_chain", "Unsupported chain.")
	SwapUnsupportedChain      = define(Invalid, "unsupported_chain", "Swaps are only supported on Stellar.")
	TrustlineUnsupportedChain = define(Invalid, "unsupported_chain", "Trustlines are only supported on Stellar.")
	InvalidAmount             = define(Invalid, "invalid_amount", "Amount must be positive.")
	RecipientRequired         = define(Invalid, "recipient_required", "Recipient is required.")
	InvalidAsset              = define(Invalid, "invalid_asset", "Asset code and issuer are required.")
	InvalidAssetPair          = define(Invalid, "invalid_asset_pair", "Choose two different assets.")
	TransactionNotFound       = define(NotFound, "transaction_not_found", "Transaction not found.")
	TransactionNotOnChain     = define(NotFound, "transaction_not_found", "Transaction not found on chain yet.")
	TrustlineExists           = define(Conflict, "trustline_exists", "Trustline already exists or is in progress.")
	MPCUnavailable            = define(Unavailable, "mpc_unavailable", "Not enough MPC nodes are online. Please try again later.")
	RecipientResolutionFailed = define(Invalid, "recipient_resolution_failed", "Unable to resolve recipient. Check the address and try again.")
	PaymentBuildFailed        = define(Invalid, "payment_build_failed", "Unable to prepare payment. Check the recipient, asset, amount and balance.")
	TrustlineBuildFailed      = define(Invalid, "trustline_build_failed", "Unable to prepare trustline. Check the asset and wallet balance.")
	SwapQuoteFailed           = define(Invalid, "swap_quote_failed", "Unable to quote this swap. Try another asset pair or amount.")
	SwapBuildFailed           = define(Invalid, "swap_build_failed", "Unable to prepare swap. Check the assets, amount and balance.")
	SignDispatchFailed        = define(Upstream, "sign_dispatch_failed", "Unable to start transaction signing.")
	KeygenDispatchFailed      = define(Upstream, "keygen_dispatch_failed", "Unable to start wallet creation.")
	FundingFailed             = define(Upstream, "funding_failed", "Unable to fund the wallet. Please try again later.")
	BalanceUnavailable        = define(Upstream, "balance_unavailable", "Unable to load balances. Please try again later.")
)
