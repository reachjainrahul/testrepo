# define the cleanup_testbed function
function cleanup_testbed() {
  ip_addr=$1
  testbed_name=$2
  GOVC_PASSWORD=$3
  echo "=== retrieve logs ==="
  scp -i id_rsa ubuntu@${ip_addr}:~/test.log ../../..

  echo "=== cleanup vm ==="
  ./destroy.sh ${testbed_name} ${GOVC_PASSWORD}

  cd ../../..
  tar zvcf test.log.tar.gz test.log
}