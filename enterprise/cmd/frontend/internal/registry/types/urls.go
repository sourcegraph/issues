package types

// ExtensionURL returns the URL path to an extension.
func ExtensionURL(extensionID string) string {
	return "/extensions/" + extensionID
}

// PublisherExtensionsURL returns the URL path to a publisher's extensions.
func PublisherExtensionsURL(isUser, isOrg bool, name string) string {
	const prefix = "/extensions/registry/publishers"
	switch {
	case isUser:
		return prefix + "/users/" + name
	case isOrg:
		return prefix + "/organizations/" + name
	default:
		return ""
	}
}
