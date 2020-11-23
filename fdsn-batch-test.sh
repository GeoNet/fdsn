#!/bin/sh
input="fdsn-test-urls.txt"
list_placeholder1="\${List1}"
list_placeholder2="\${List2}"
list1="ABAZ,AKCZ,ALRZ,AMCZ,ANWZ,ARAZ,ARCZ,ARHZ,AWAZ,BHHZ,CAW,CKHZ,CMWZ,CNGZ,CPWZ,CRSZ,DREZ,DUWZ,DVHZ,EDRZ,EPAZ,ETAZ,GCSZ,HBAZ,HLRZ,HOWZ,HRRZ,HSRZ,KAHZ,KARZ,KATZ,KBAZ,KIW,KMRZ,KRHZ,KRVZ,KUTZ,KWHZ,LIRZ,LREZ,MARZ,MBAZ,MCHZ,MHCZ,MHEZ,MHGZ,MKRZ,MOVZ,MRHZ,MRNZ,MSWZ,MTHZ,MTVZ,MTW,MUGZ,MYRZ,NBEZ,NEZ,NGRZ,NGZ"
list2="WEL,WEL,WEL,WEL,WEL,WEL,WEL,WEL"
# firstly run through the test (should be against service.geonet.org.nz)
while IFS= read -r line
do
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
        echo "Error $line expect $res got result $r"
        exit 1
    fi
  fi
done < "$input"

# now do nrt
service="service.geonet.org.nz"
service_nrt="service-nrt.geonet.org.nz"
sample_date="2018-05-15"
nrt_date=$(date -v-1d +'%Y-%m-%d') # 1 day ago
echo "NRT test date: $nrt_date"

while IFS= read -r line
do
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
    line=${line//$sample_date/$nrt_date} # replace date to 1 day ago
    echo "$line"
    r=$(curl -f --write-out %{http_code} -s --output /dev/null "$line")
    if ! [ $? -eq 0 ] || ! [ $r -eq ${res} ] ; then
        echo "Error $line expect $res got result $r"
        exit 1
    fi
  fi
done < "$input"
