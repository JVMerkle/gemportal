package gemini

// Gemini status codes as defined in the Gemini spec Appendix 1.
const (
	StatusInput          = 10
	StatusSensitiveInput = 11

	StatusSuccess                              = 20
	StatusSuccessEndOfClientCertificateSession = 21

	StatusRedirect          = 30
	StatusRedirectTemporary = 30
	StatusRedirectPermanent = 31

	StatusTemporaryFailure = 40
	StatusUnavailable      = 41
	StatusCGIError         = 42
	StatusProxyError       = 43
	StatusSlowDown         = 44

	StatusPermanentFailure    = 50
	StatusNotFound            = 51
	StatusGone                = 52
	StatusProxyRequestRefused = 53
	StatusBadRequest          = 59

	StatusClientCertificateRequired     = 60
	StatusTransientCertificateRequested = 61
	StatusAuthorisedCertificateRequired = 62
	StatusCertificateNotAccepted        = 63
	StatusFutureCertificateRejected     = 64
	StatusExpiredCertificateRejected    = 65
)

var statusText = map[int]string{
	StatusInput:          "Input",
	StatusSensitiveInput: "Sensitive Input",

	StatusSuccess: "Success",
	StatusSuccessEndOfClientCertificateSession: "End Of Client Certificate Session",

	StatusRedirectTemporary: "Temporary Redirect",
	StatusRedirectPermanent: "Permanent Redirect",

	StatusTemporaryFailure: "Temporary Failure",
	StatusUnavailable:      "Unavailable",
	StatusCGIError:         "CGI Error",
	StatusProxyError:       "Proxy Error",
	StatusSlowDown:         "Slow Down",

	StatusPermanentFailure:    "Permanent Failure",
	StatusNotFound:            "Not Found",
	StatusGone:                "Gone",
	StatusProxyRequestRefused: "Proxy Request Refused",
	StatusBadRequest:          "Bad Request",

	StatusClientCertificateRequired:     "Client Certificate Required",
	StatusTransientCertificateRequested: "Transient Certificate Requested",
	StatusAuthorisedCertificateRequired: "Authorised Certificate Require",
	StatusCertificateNotAccepted:        "Certificate Not Accepted",
	StatusFutureCertificateRejected:     "Future Certificate Rejected",
	StatusExpiredCertificateRejected:    "Expired Certificate Rejected",
}

// StatusText returns a text for the Gemini status code. It returns the empty
// string if the code is unknown.
func StatusText(code int) string {
	return statusText[code]
}
