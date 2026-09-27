package language

import "testing"

func TestDetect(t *testing.T){
	tests:=map[string]string{
		"main.go":"Go","app.py":"Python","component.TSX":"TypeScript",
		"expert.mq5":"MQL5","Program.cs":"C#","script.ps1":"PowerShell",
		"native.cpp":"C++","header.h":"C/C++ Header","unknown.asset":"",
	}
	for path,want:=range tests{
		got,ok:=Detect(path)
		if want==""{
			if ok{t.Fatalf("Detect(%q) unexpectedly returned %q",path,got.Name)}
			continue
		}
		if !ok||got.Name!=want{t.Fatalf("Detect(%q)=%q,%v; want %q,true",path,got.Name,ok,want)}
	}
}
