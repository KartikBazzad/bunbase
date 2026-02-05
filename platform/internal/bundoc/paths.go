package bundoc

// Path shapes for platform → bundoc. Do not change without checking tenant-auth and ProxyHandler.
//
// - Tenant-auth: calls bundoc directly (not via platform); uses .../databases/(default)/documents/...
// - ProxyHandler (API-key DB): uses /databases/{projectID}/documents/{collection}/{docID} — do not use BundocDBPath.
// - DeveloperProxyHandler (dashboard): uses BundocDBPath below; get/update/delete use .../documents/{collection}/{docId}.
//
// ProxyRequest builds: BaseURL + "/v1/projects/" + projectID + path.
const (
	// BundocDBPath is used only by DeveloperProxyHandler (dashboard). Must match bundoc parseProjectAndPathSuffix (documents at index 5).
	BundocDBPath = "/databases/(default)"
)
