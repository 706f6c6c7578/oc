If you have a Raspberry Pi or VPS, running an Onion Courier node is very simple.

1. Open port 8088 in your firewall.

2. Create an additional Tor Hidden Service and start it.
   Add a user, named 'ocn'.

3. Copy the hostname file to your Desktop.

4. Download oc_node_server.go, minicrypt and Eurasia hasher.

https://github.com/706f6c6c7578/oc
https://github.com/706f6c6c7578/minicrypt
https://github.com/706f6c6c7578/eh

5. Create a key pair with minicrypt

   $ minicrypt -g

6. Create your node password/id

   Use the data in the hostname file and use a nickname with 'eh':

   $ echo -n 'nickname:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.onion:8088' | eh -ripemd160

   This results in a ripemd160 hash wich you must insert in line 27 of the Go code:
   
   serverPassword = "secretPassword" // Set your desired server password here
                     ^^^^^^^^^^^^^^

7. Create a folder, named 'ocn' on your Desktop and put the edited oc_node_server.go in it.

8. Compile: 'go mod init ocn', 'go mod tidy', 'go build -ldflags "-s -w"'.

9. Upload the binary and the previously generated private.pem to your VPS Account 'ocn'

10. Start ocn: $ ./ocn -s /path/to/your/private.pem.

    Before set permission 600 for private.pem and press CNTRL-Z when
    the server is running and after that type 'bg'. Then log out.

11. Publish your data, in a.p.a-s, or other places, which must look like this:

    nickname:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.onion:8088 ripemd160 hash

    and the public.pem of your node.

It will be then added to the public nodes list, which you can find here:

https://github.com/706f6c6c7578/oc/blob/main/README_public_nodes.txt
