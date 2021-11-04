#!/bin/sh
input="fdsn-test-urls.txt"
list_placeholder1="\${List1}"
list_placeholder2="\${List2}"
list1="ABAZ,AKCZ,ALRZ,AMCZ,ANWZ,ARAZ,ARCZ,ARHZ,AWAZ,BHHZ,CAW,CKHZ,CMWZ,CNGZ,CPWZ,CRSZ,DREZ,DUWZ,DVHZ,EDRZ,EPAZ,ETAZ,GCSZ,HBAZ,HLRZ,HOWZ,HRRZ,HSRZ,KAHZ,KARZ,KATZ,KBAZ,KIW,KMRZ,KRHZ,KRVZ,KUTZ,KWHZ,LIRZ,LREZ,MARZ,MBAZ,MCHZ,MHCZ,MHEZ,MHGZ,MKRZ,MOVZ,MRHZ,MRNZ,MSWZ,MTHZ,MTVZ,MTW,MUGZ,MYRZ,NBEZ,NEZ,NGRZ,NGZ"
list2="WEL,WEL,WEL,WEL,WEL,WEL,WEL,WEL"

if [ -z "$1" ]; then
# firstly run through the test (should be against service.geonet.org.nz)
linenum=0
while IFS= read -r line
do
  linenum=$((linenum + 1))
  if ! [[ $line = \#* ]] && ! [[ -z $line ]] ; then
    tk=(${line//;;/ })
    if (( ${#tk[*]} > 1 )) ; then
      res="${tk[1]}"
    else
      res="200"
    fi
    line=${tk[0]/$list_placeholder1/$list1}  # replace stations
    line=${line/$list_placeholder2/$list2}  # replace stations
    echo "expecting $res for $line"
    r=$(curl -f --write-out %{http_code} -s --output /dev/null "$line")
    if ! [ $? -eq 0 ] || ! [ $r -eq ${res} ] ; then
        echo "L$linenum: Error $line expect $res got result $r"
        exit 1
    fi
  fi
done < "$input"
fi

# now do nrt
service="https://service.geonet.org.nz"

if [ -z "$1" ]; then
service_nrt="https://service-nrt.geonet.org.nz"
else
service_nrt="$1"
fi 
sample_date="2018-05-15"
sample_tstart="23:45:00"
sample_tend="23:45:10"
# Note1 : this date/time string replacing won't work when start/end window is crossing midnight
# Note2 : For OSX replace `-d "60 minutes ago"` with `-v-60M`, and -v-30M for end time.
# Note3 : The time range (60 minutes ago) is for NRT service. If it's FDSN archive then 10080 minutes ~ 10075 minutes ago.
nrt_date=$(TZ="GMT0" date -d "60 minutes ago" +'%Y-%m-%d')
nrt_tstart=$(TZ="GMT0" date -d "60 minutes ago" +'%H:%M:%S') #start: 60 min ago
nrt_tend=$(TZ="GMT0" date -d "30 minutes ago" +'%H:%M:%S') #end: 30 min ago
echo "NRT test date: $nrt_date $nrt_tstart to $nrt_tend"

linenum=0
while IFS= read -r line
do
  linenum=$((linenum + 1))
  if ! [[ $line = \#* ]] && ! [[ -z $line ]] ; then
    tk=(${line//;;/ })
    if (( ${#tk[*]} > 1 )) ; then
      res="${tk[1]}"
    else
      res="200"
    fi
    line=${tk[0]/$list_placeholder1/$list1}  # replace stations
    line=${line/$list_placeholder2/$list2}  # replace stations
    line=${line/$service/$service_nrt}  # replace server name to nrt
    line=${line//$sample_date/$nrt_date} # replace date today
    line=${line//$sample_tstart/$nrt_tstart} # replace starttime to 20 mins ago
    line=${line//$sample_tend/$nrt_tend} # replace endtime to 15 mins ago
    echo "$line"
    r=$(curl -f --write-out %{http_code} -s --output /dev/null "$line")
    if ! [ $? -eq 0 ] || ! [ $r -eq ${res} ] ; then
        echo "L$linenum: Error $line expect $res got result $r"
        exit 1
    fi
  fi
done < "$input"
