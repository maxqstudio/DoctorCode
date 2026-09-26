package language

import (
	"path/filepath"
	"strings"
)

type Definition struct {
	Name string
	Extensions []string
}

var definitions=[]Definition{
	{Name:"C",Extensions:[]string{".c"}},
	{Name:"C/C++ Header",Extensions:[]string{".h"}},
	{Name:"C++",Extensions:[]string{".cc",".cpp",".cxx",".hh",".hpp",".hxx"}},
	{Name:"C#",Extensions:[]string{".cs"}},
	{Name:"Dart",Extensions:[]string{".dart"}},
	{Name:"Go",Extensions:[]string{".go"}},
	{Name:"Java",Extensions:[]string{".java"}},
	{Name:"JavaScript",Extensions:[]string{".js",".jsx",".mjs",".cjs"}},
	{Name:"Kotlin",Extensions:[]string{".kt",".kts"}},
	{Name:"MQL5",Extensions:[]string{".mq5",".mqh"}},
	{Name:"PHP",Extensions:[]string{".php"}},
	{Name:"PowerShell",Extensions:[]string{".ps1",".psm1",".psd1"}},
	{Name:"Python",Extensions:[]string{".py",".pyi"}},
	{Name:"Ruby",Extensions:[]string{".rb"}},
	{Name:"Rust",Extensions:[]string{".rs"}},
	{Name:"Shell",Extensions:[]string{".sh",".bash",".zsh"}},
	{Name:"Swift",Extensions:[]string{".swift"}},
	{Name:"TypeScript",Extensions:[]string{".ts",".tsx",".mts",".cts"}},
	{Name:"Zig",Extensions:[]string{".zig"}},
}

func Definitions() []Definition {
	out:=make([]Definition,len(definitions))
	copy(out,definitions)
	return out
}

func Detect(path string)(Definition,bool){
	ext:=strings.ToLower(filepath.Ext(path))
	for _,def:=range definitions{
		for _,candidate:=range def.Extensions{
			if ext==candidate{return def,true}
		}
	}
	return Definition{},false
}
