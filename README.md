
# aria2helper

this is an aria2 start helper.

* auto install and upgrade aria2(for windows).
* auto config aria2.conf.
* auto update bt tracker list before start aria2.

## Requirements

golang version >= 1.12 [downlad link](https://golang.org/dl/)
## Usage

### Windows
1. download code and build it
```powershell
go get github.com/Art2Cat/aria2helper
go build .
```
1. Move aria2helper.exe to the directory(e.g D:\\aria2) where you want to install aria2.

2. Double click aria2helper.exe to run. Or you can run start.ps1 to hide the cmd window.`PowerShell required`

### Linux/Unix

```bash
# download the code
go get github.com/Art2Cat/aria2helper
go build .

# Move aria2helper to the directory(e.g ~/aria2) where you installed aria2.
cp ./aria2helper ~/aria2

cd ~/aria2
# grant execute permission to aria2helper
sudo chmod u+x aria2helper

# run aria2helper in background
./aria2helper &
```
## Note
the default download directory for windows is `D:\Downloads`, for Linux/Unix is `~/Downloads`.

## License
Copyright (c) Rorschach Huang. All rights reserved. Please read the [LICENSE](LICENSE) for more details.