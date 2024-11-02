## A message from Alice to Bob

$ oc_client -d data.txt -f msg.txt

where data.txt contains the onionURL,
port and password, in the following format:

onionURL:port password
LF or CRLF (must be one line only, otherwise
a format error appears)

## A message to the Usenet Group alt.test

Subject: Test
Newsgroups: alt.test
MIME-Version: 1.0
Content-Type: text/plain; charset=us-ascii
Content-Transfer-Encoding: 7bit

Hello world!

## A reply message to the above Usenet posting

Subject: Re: Test
References: <Message-ID of original posting>
Newsgroups: alt.test
MIME-Version: 1.0
Content-Type: text/plain; charset=us-ascii
Content-Transfer-Encoding: 7bit

> Hello world!

Hi

## An email, via a Guard node and Middleman

data.txt must contain as first hop the
onionURL:port password of the Guard node.

The message then needs a prepended header
of the middleman.

X-OC-To: onionURl:port password of middleman
To: bob@example.org
Subject: Hello

Hi Bob!

Best Alice

After that the message  needs to be encrypted
with the public key of the Guard node, done with
minicrypt and send to the guard node in data.txt.
