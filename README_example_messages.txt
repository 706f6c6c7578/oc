## A message from Alice to Bob:

$ oc_client -u Alice -d data.txt -f msg.txt

where data.txt contains the onionURL,
port and password, in the following format:

onionURL:port password
LF or CRLF (must be one line only, otherwise
a format error appears)

## A message to the Usenet Group alt.test

data.txt contains the mailer data:

To: mail2news@dizum.com
Subject: Test
Newsgroups: alt.test
MIME-Version: 1.0
Content-Type: text/plain; charset=us-ascii
Content-Transfer-Encoding: 7bit

Hello world!

## A reply message to the above Usenet posting

To: mail2news@dizum.com
Subject: Re: Test
References: <Message-ID of original posting>
Newsgroups: alt.test
MIME-Version: 1.0
Content-Type: text/plain; charset=us-ascii
Content-Transfer-Encoding: 7bit

> Hello world!

Hi

## An email, via nodes:

data.txt must contain as first hop the
onionURL:port password of the first hop.

The message with headers needs to be encrypted
with minicrypt and the nodes public key.

Chaining of nodes is done by re-encrypting the
encrypted message with the next hop public key.

The final message then needs a prepended header
of the first node in the chain.

X-OC-To: onionURl:port password of node_1, node_2, etc.
X-OC-To: onionURL:port password of mailer
To: bob@example.org
Subject: Hello

Hi Bob!

Best Alice
