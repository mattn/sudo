# sudo for windows

## Usage

```
C:\>sudo cmd /c dir
```

Then, you'll see the UAC dialog.

## Tutorials

### Display contents of file which can't access from you

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

## License

MIT

## Author

Yasuhiro Matsumoto (a.k.a. mattn)
