# Toy CQRS + EventSourcing

As the name of the repo implies, this is a toy implementation, 
probably wrong and lacking, of CQRS pattern with EventSourcing. 

Try not to use read too much into this, this was my first attempt 
in implementing something like this and only serves the purpose
of me learning something, **not as an example of how to do things**. 
I am not an expert on CQRS and ES and number of technologies I have 
used here are used because it was fun, not because I know them or 
are best tool for the job. Since learning and having fun while doing
this was main motivator, entire project is hugely over-engineered. 

## Start
To start locally (the only way I have tested this), you need
[Go](https://golang.org) and [Docker](https://docker.com) installed.

Go version I used is `1.16`, it might work with earlier versions, but 
I did not try it. 

Executing `run.sh` script will build the project and start
`docker-compose` with all needed containers. API is exposed on
`8001` by default. 

Additional containers will take other ports as well, mostly in range
of 8xxx, except well known ports (e.g. for postgres).

## TODO
Many things are missing, I am not sure how much time I will have
to work on this or what kind of inspiration will come, so I am
not creating a fixed TODO list. 

In general, I would like to work on ensuring idempotency, 
conflicts resolution, error reporting, performance monitoring, etc.

## Contributions
Contributing to this project is contributing to my knowledge. 
So, if you detect any wrong patterns (or anti-patterns), area 
for improvements, etc., do reach out (any way you can, opening 
an issue on this repo is completely fine). 

I am open to accepting pull requests, but main point is for me to
learn something new and play around, so if someone else is implementing
it, I am missing out on the main goal :). That being said, if you see 
something minor to improve, feel free to send a Pull Request, but
it is probably best to open an issue to discuss it first. 

## Author
Bojan Delic <bojan@delic.in.rs>
