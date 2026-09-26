package scanner

import (
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/maxqstudio/DoctorCode/internal/language"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

var ignoredDirectories=map[string]struct{}{
	".git":{},".hg":{},".svn":{},".idea":{},".vscode":{},
	"node_modules":{},"vendor":{},"dist":{},"build":{},"target":{},
	".venv":{},"venv":{},"__pycache__":{},
}

func Scan(root string)(model.ScanResult,error){
	absoluteRoot,err:=filepath.Abs(root)
	if err!=nil{return model.ScanResult{},err}
	counts:=map[string]int{}
	total:=0
	recognized:=0
	err=filepath.WalkDir(absoluteRoot,func(path string,entry fs.DirEntry,walkErr error)error{
		if walkErr!=nil{return walkErr}
		if entry.IsDir(){
			if path!=absoluteRoot{
				if _,skip:=ignoredDirectories[entry.Name()];skip{return filepath.SkipDir}
			}
			return nil
		}
		total++
		if def,ok:=language.Detect(path);ok{
			counts[def.Name]++
			recognized++
		}
		return nil
	})
	if err!=nil{return model.ScanResult{},err}
	names:=make([]string,0,len(counts))
	for name:=range counts{names=append(names,name)}
	sort.Strings(names)
	languages:=make([]model.LanguageSummary,0,len(names))
	for _,name:=range names{languages=append(languages,model.LanguageSummary{Name:name,Files:counts[name]})}
	return model.ScanResult{Root:absoluteRoot,Files:total,RecognizedFiles:recognized,Languages:languages},nil
}
