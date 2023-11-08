package nntpcli

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maybeId(cmd, id string) string {
	if len(id) > 0 {
		return cmd + " " + id
	}
	return cmd
}

func ProviderName(host, username string) string {
	return host + "-" + username
}
