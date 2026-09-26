package toolchain

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

type Candidate struct {
	Language string
	Tools []string
	Manifests []string
}

var candidates=[]Candidate{
	{Language:"C",Tools:[]string{"cc","gcc","clang"},Manifests:[]string{"CMakeLists.txt","Makefile"}},
	{Language:"C++",Tools:[]string{"c++","g++","clang++"},Manifests:[]string{"CMakeLists.txt","Makefile"}},
	{Language:"C#",Tools:[]string{"dotnet"},Manifests:[]string{"*.sln","*.csproj"}},
	{Language:"Dart",Tools:[]string{"dart","flutter"},Manifests:[]string{"pubspec.yaml"}},
	{Language:"Go",Tools:[]string{"go"},Manifests:[]string{"go.mod","go.work"}},
	{Language:"Java",Tools:[]string{"javac","mvn","gradle"},Manifests:[]string{"pom.xml","build.gradle","build.gradle.kts"}},
	{Language:"JavaScript",Tools:[]string{"node","npm"},Manifests:[]string{"package.json"}},
	{Language:"Kotlin",Tools:[]string{"kotlinc","gradle"},Manifests:[]string{"build.gradle.kts","settings.gradle.kts"}},
	{Language:"MQL5",Tools:[]string{"metaeditor64.exe","metaeditor.exe"}},
	{Language:"PHP",Tools:[]string{"php"},Manifests:[]string{"composer.json"}},
	{Language:"PowerShell",Tools:[]string{"pwsh","powershell.exe"}},
	{Language:"Python",Tools:[]string{"python3","python"},Manifests:[]string{"pyproject.toml","requirements.txt","setup.py"}},
	{Language:"Ruby",Tools:[]string{"ruby","bundle"},Manifests:[]string{"Gemfile"}},
	{Language:"Rust",Tools:[]string{"cargo","rustc"},Manifests:[]string{"Cargo.toml"}},
	{Language:"Shell",Tools:[]string{"bash","sh"}},
	{Language:"Swift",Tools:[]string{"swift"},Manifests:[]string{"Package.swift"}},
	{Language:"TypeScript",Tools:[]string{"tsc","node","npm"},Manifests:[]string{"tsconfig.json","package.json"}},
	{Language:"Zig",Tools:[]string{"zig"},Manifests:[]string{"build.zig"}},
}

func Candidates()[]Candidate{
	out:=make([]Candidate,len(candidates))
	copy(out,candidates)
	return out
}

func Detect(root string)[]model.ToolchainStatus{
	absoluteRoot,err:=filepath.Abs(root)
	if err!=nil{absoluteRoot=root}
	out:=make([]model.ToolchainStatus,0,len(candidates))
	for _,candidate:=range candidates{
		status:=model.ToolchainStatus{Language:candidate.Language}
		if len(candidate.Tools)>0{status.Tool=candidate.Tools[0]}
		for _,tool:=range candidate.Tools{
			if path,lookErr:=exec.LookPath(tool);lookErr==nil{
				status.Tool=tool;status.Available=true;status.Path=path;break
			}
		}
		for _,pattern:=range candidate.Manifests{
			matches,globErr:=filepath.Glob(filepath.Join(absoluteRoot,pattern))
			if globErr!=nil{continue}
			for _,match:=range matches{
				if info,statErr:=os.Stat(match);statErr==nil&&!info.IsDir(){status.Manifests=append(status.Manifests,filepath.Base(match))}
			}
		}
		sort.Strings(status.Manifests)
		out=append(out,status)
	}
	return out
}
