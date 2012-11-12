wb
==

A simple tool for a "web benchmark". Call it without any parameter to see the usage:

    Usage of ./wb <url>:
     -c=1: Number of concurrent requests to make
     -n=1: Number of requests to perform
     -r=false:  Reuse HTTP Client in every worker
     -v=0: Show info while running
     -y=false: Create new HTTP Client for every request

You can specify the number of overall requests, the number of parallel requests an some sort of verbosity. 
If verbosity is 0, only a summary is printed at the end. Specify "1" if you want every 100 Request to be 
summarized and specify "2" if you want every Request to be printed.

If you specify a "-r" option all concurrent Request use the same HTTP Client and use KeepAlive, so there will
be only one physical connection with many HTTP Connections. If you don't reuse the Client, every concurrent
worker use it's own connection (with KeepAlive), so there will be as many connections as parallel requests
are speicified.

The "-y" option creates a new client for every request. So be careful when using this option, you will need many resources with this option.