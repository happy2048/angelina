package client

var InitTemplate = `{
    "AuthFile": "",
    "ReferVolume": "",
    "DataVolume" : "",
    "GlusterEndpoints": "",
    "Namespace": "",
    "ScriptUrl": "",
    "OutputBaseDir": "",
	"StartRunCmd": "",
	"ControllerContainer": ""
}`

var ConfigTemplate = `{
	"input-directory": "",
	"glusterfs-entry-directory": "",
	"sample-name": "",
	"redis-address":"",
	"template-env": ["",""],
	"pipeline-template-name": "",
	"force-to-cover": "no"
}`

var PipelineTemplate = `{
	"pipeline-name": "",
	"pipeline-description": "",
	"pipeline-content": {
		"refer" : {
			"": "",
			"": ""
		},
		"input": ["",""],
		"params": {
			"": "",
			"": ""
		},
		"step1": {
        	"pre-steps": ["",""],
        	"container": "",
			"command-name": "",
        	"command": ["",""],
        	"args":["",""],
        	"sub-args": [""]
		},
		"step2": {
        	"pre-steps": ["",""],
        	"container": "",
			"command-name": "",
        	"command": ["",""],
        	"args":["",""],
        	"sub-args": [""]
		}
	}
}`












