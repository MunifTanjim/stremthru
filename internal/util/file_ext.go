package util

var FileExtVideo = func() *Set[string] {
	s := NewSet[string]()
	for _, ext := range []string{
		".3g2",
		".3gp",
		".amv",
		".asf",
		".avi",
		".drc",
		".f4a",
		".f4b",
		".f4p",
		".f4v",
		".flv",
		".gif",
		".gifv",
		".m2ts",
		".m2v",
		".m4p",
		".m4v",
		".mk3d",
		".mkv",
		".mng",
		".mov",
		".mp2",
		".mp4",
		".mpe",
		".mpeg",
		".mpg",
		".mpv",
		".mxf",
		".nsv",
		".ogg",
		".ogm",
		".ogv",
		".qt",
		".rm",
		".rmvb",
		".roq",
		".svi",
		".ts",
		".webm",
		".wmv",
		".yuv",
	} {
		s.Add(ext)
	}
	return s
}()

var FileExtSubtitle = func() *Set[string] {
	s := NewSet[string]()
	for _, ext := range []string{
		".srt",
		".ass",
		".ssa",
		".sub",
	} {
		s.Add(ext)
	}
	return s
}()
