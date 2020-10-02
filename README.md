# sudo

sudo for windows

[![Build status](https://ci.appveyor.com/api/projects/status/xyxiefgm9giyags3?svg=true)](https://ci.appveyor.com/project/mattn/sudo)

## Usage

```
C:\>sudo cmd /c dir
```

Then, you'll see the UAC dialog.

## Tutorials/education

### Display contents of file which can't access from you desktop

```
sudo cmd /c type secret-file.txt > accessible-file.txt
```

### Pipe from/to stream

```
echo 123 | sudo my-command.exe | more
```

### Change IP address

```
sudo netsh interface ip add address "Local Area Connection" 33.33.33.33 255.255.255.255
```

### Edit hosts file

```
sudo notepad c:\windows\system32\drivers\etc\hosts
```

### Create admin's console

```
sudo
```

## Installation

```
go get github.com/mattn/sudo
```

requirement go1.8 or later.

Or download from [release](https://github.com/mattn/sudo/releases) tab.

## License

MIT

## Author

Yasuhiro Matsumoto (a.k.a. mattn)
