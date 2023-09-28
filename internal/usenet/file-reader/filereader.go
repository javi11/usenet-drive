package filereader


if isNzbFile(name) {
	// If file is a nzb file return a custom file that will mask the nzb
	return NewNZBFileInfo(name, name, fs.log, fs.nzbLoader)
}

originalName := getOriginalNzb(name)
if originalName != nil {
	// If the file is a masked call the original nzb file
	return NewNZBFileInfo(*originalName, name, fs.log, fs.nzbLoader)
}