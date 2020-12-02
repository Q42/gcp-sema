export MOCK_SEMA=1
gcp-sema render dummy --secrets literal=test.txt=value --secrets literal=foo.txt=bar
