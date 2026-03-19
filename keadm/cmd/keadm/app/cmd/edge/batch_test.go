package edge

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const testKnownHosts = `# Test Data
1.1.1.1,example.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDQyTpYG91GW/bdnOmy1xPRwHd6xGbR5jaZhSWccOw6H6nOgYYHD0KK2aNpE56nBSWOR5NbBguJUch7pvCLD0OJRJA/WgAe8cDBdc6SZb4j39JB5bx0uNtaYVb0p2YmCkp6faF017JZveaw8bINSNowSyNOn7bjHO3Un74a6wEnt8In3Xs379BmJgJrvUYzophEcXUZ1PdquUQSaTZYnPTmmAp/4GbzkXOcAwOiAWet2tVZLrj/3DvTonNnJ8xWkz1UKMerDlfb9JHC98efhWTd25+abYAezEzWl5snzW5esGFbOZclGaIZMV/aXw8kbF7Q7PrbvXrYypCboc1m5rAj48QQSVEKCWc4xvt1EIFDD/8wBVQJVn4aXxmQBEHhdDQLxY05GhUtBfN+cMrwm3xkrWg/draKIPH2Zmdhn3lcgcf5+1Vuy3E5pJKLEl3NDH5XdvRsGtExpN8695thOuocd7ouKvYoK7inVQ/PWO1gfPD3txnBhSQ/ym1gsZb/K2U=
`

func TestParseKnownHosts(t *testing.T) {
	const filename = "known_hosts"
	err := os.WriteFile(filename, []byte(testKnownHosts), 0644)
	require.NoError(t, err)
	defer os.Remove(filename)

	hosts, err := parseKnownHosts(filename)
	require.NoError(t, err)
	require.Len(t, hosts, 2)
}
