#~/bin/bash

badfmtfilecnt=$( gofmt -l . | grep -v "emoteabac\/Godeps\/" | wc -l )
echo $badfmtfilecnt go files not matching gofmt standard
exit $badfmtfilecnt




