package webdav

type contextKey string

func (c contextKey) String() string {
	return "webdav context key " + string(c)
}

const reqContentLengthKey = contextKey("reqContentLength")
