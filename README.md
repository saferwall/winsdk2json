# winsdk2json

sdk2json is a go package that parses the Windows SDK (function prototypes, structures, unions) to a friendlier format like JSON. The purpose of this tool was initially for the API hooking module (of saferwall sandbox) to allow an implementation of a generic hook handler that does not require knowledge of specific API prototype to know how to log them.

`winsdk2json` relies on a [C compiler frontend](https://gitlab.com/cznic/cc) for parsing the C headers and convert them to a higher level objects. Those objects can be serialized formats like JSON or YAML.

Here is an example:

```json
{
 "advapi32.dll": {
  "ControlService": {
   "callconv": "WINAPI",
   "name": "ControlService",
   "retVal": "BOOL",
   "params": [
    {
     "anno": "_In_",
     "type": "SC_HANDLE",
     "name": "hService"
    },
    {
     "anno": "_In_",
     "type": "DWORD",
     "name": "dwControl"
    },
    {
     "anno": "_Out_",
     "type": "LPSERVICE_STATUS",
     "name": "lpServiceStatus"
    }
   ]
  },
```

## Usage

You can download the JSON files that contains all the definitions of the Win32 APIs in the releases page. If however, you intend to generate those definitions on another custom that fits your needs, please follow the instructions below:

Clone the repo:

```shell
git clone https://github.com/saferwall/winsdk2json.git
cd winsdk2json
git submodule init
git submodule update
```

Build & run"

```shell
go build cli.go
.\cli.exe -h

WinSdk2JSON - a tool to parse the Windows Win32 SDK into JSON format.
For more details see the github repo at https://github.com/saferwall/winsdk2json

Usage:
  winsdk2json [flags]
  winsdk2json [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  parse       Walk through the Windows SDK and parse the Win32 headers
  version     Version number

Flags:
  -h, --help   help for winsdk2json
```

This package tries to stay up to date with the latest [Windows SDK](https://developer.microsoft.com/en-us/windows/downloads/windows-sdk/), they are copied into the `winsdk/` folder. However, feel free to use an SDK version that is not included in this repo. The `parse` command takes a `i` argument that points to the Windows SDK headers. For example: `C:\\Program Files (x86)\\Windows Kits\\10\\Include\\10.0.19041.0\\`.

## Lessons Learned

- SAL annotations
- Some Windows APIs does not have a ANSI alternative: i.e OpenMutexW
- API like `CreateToolhelp32Snapshot` does not have annotations.

## References

- https://github.com/microsoft/win32metadata
- https://github.com/BehroozAbbassi/sdkffi