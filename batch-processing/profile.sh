max=30
for i in `seq 0 $max`
do
    curl http://localhost:6060/debug/pprof/heap > ./outputs/profile/heap.$i.pprof
    sleep 2
done