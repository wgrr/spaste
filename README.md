spaste sourcehut paste service client.

Usage:

     go get -u git.sr.ht/~wgr/spaste

     spase (-e | -t) [ file... ]

## Example

     $ spaste -e $(gpg password_file) file
     https://paste.sr.ht/blob/9801739daae44ec5293d4e1f53d3f4d2d426d91c
     $

     $ curl -L example.com | spaste -t 42424242
     https://paste.sr.ht/blob/9801739daae44ec5293d4e1f53d3f4d2d426d91c
     $

## Known issues:
     missing GET method endpoints (cat, ls)
     missing DELETE method endpoints (rm)
     can't do multiple files into a single paste
