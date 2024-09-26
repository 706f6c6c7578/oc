## Tor-Enabled data exchange with Client oc_client.go and Server oc_server.go

## Client oc_client.go
This client allows you to securely upload files to a server
using either the Tor network (for enhanced privacy) or clearnet.  
It supports various options for flexibility and ease of use.

## Features

- Upload data through Tor network (default) or clearnet
- Optional username support
- Use of a data file for server address:port and password

## Requirements

- Go programming language
- `golang.org/x/net/proxy` package (for Tor support)

To install the required package, run:

go get golang.org/x/net/proxy

To compile oc_client.go and oc_server.go use:

$ go build -ldflags "-s -w" oc_client.go  
$ go build -ldflags "-s -w" oc_client.go


- Tor Expert Bundle
- `https://www.torproject.org/download/tor/`


## Usage

### Basic Usage

Start tor.exe from the Tor Expert Bundle, then run the compiled oc_client.go program.

$ oc_client [-u username] [-d datafile] [-clearnet] -f filename <server_address:port> <password>

### Using a Data File

$ oc_client [-u username] -d <data_file> [-clearnet] -f filename

The data file should contain the server address:port and password, separated by a space, on a single line, ending with LF or CRLF.  
Multiple entries are supported.  
You can put comment lines, starting with #, in your data_file too.  
Please note:  The last entry in your data_file must be a single LF or CRLF and not more.

## Options

- `-u [username]`: Specify an optional username
- `[-clearnet`]: Use clearnet instead of Tor network
- `-f <filename>`: Specify the filename to be send, which is used on the server
- `-d <data_file>`: Use a data file containing server address:port and password

## Examples

1. Upload data through Tor:

$ oc_client -f myfile.txt example.com:8080 mypassword


2. Upload data using clearnet with a username:

$ oc_client -u myusername -clearnet example.com:8080 mypassword -f myfile.txt


3. Upload data using a data file:

$ oc_client -d server_data.txt -f myfile.txt


## Notes

- By default, the client uses the Tor network. Make sure you have Tor
  running on your system (usually on port 9050).
- When using the `-clearnet` option, your IP address will be visible to the server.

## Security Considerations

- Be cautious when uploading sensitive files and consider using encryption before upload.
- The Tor network provides anonymity but may be slower than clearnet uploads.


## Server oc_server.go

## Requirements

- Tor Browser Bundle
- `https://www.torproject.org/`

## Usage

On Windows open the torrc file located, for example, at:

C:\Tor Browser\Browser\TorBrowser\Data\Tor\

and add the following two lines:

HiddenServiceDir C:\Tor Browser\Browser\TorBrowser\Data\Tor\hidden_service  
HiddenServicePort 8080 127.0.0.1:8080

Restart Tor Browser and your Tor Hidden Service folder will be created at
the same location your torrc file resides. Look at the file 'hostname' which
contains the .onion URL for your Tor Hidden Service and give the address to
your friends.

Edit the password in line 13 in the source code and compile.

Now run the compiled oc_server.go with the -p parameter, for the file path files
should be stored.

Thats all.

You now have Tor and Tor Browser running to exchange files with your
friends, as long as Tor Browser with oc_server.go is running.

## Note

You can omit the usage of Tor Browser and instead use only Tor-Expert-Bundle
and put the torrc file in the tor folder and start tor.exe with the -f parameter,
to use the torrc file.

If you put the torrc file into the tor folder, of Tor-Expert-Bundle, the torrc file may look like this:

HiddenServiceDir hidden_service  
HiddenServicePort 8080 127.0.0.1:8080

If you like the idea of point to point communication, without third-party
servers involved, consider a donation to the following Bitcoin address:

bc1qhgek8p5qcwz7r6502y8tvenkpsw9w5yafhatxk

This project is dedicated to Alice and Bob.

