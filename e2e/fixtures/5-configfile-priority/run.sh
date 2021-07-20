# Should use default location
export MOCK_SEMA=1
gcp-sema render dummy
echo "---"

# Should give preference to environment variable over default
export MOCK_SEMA=1
export SEMA_CONFIG=alternative-config.yml
gcp-sema render dummy
echo "---"

# Should give preference to cli arg over environment variable
export MOCK_SEMA=1
export SEMA_CONFIG=alternative-config.yml
gcp-sema render --config=cli-arg-config.yml dummy
