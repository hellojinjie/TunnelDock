package sshclient

type Command struct {
	Executable string
	Args       []string
}

func BuildTunnelCommand(executable, runtimeConfig, forwardSpec, hostAlias string) Command {
	return Command{
		Executable: executable,
		Args: []string{
			"-F", runtimeConfig,
			"-N", "-T", "-n",
			"-L", forwardSpec,
			"-o", "ExitOnForwardFailure=yes",
			"-o", "ServerAliveInterval=15",
			"-o", "ServerAliveCountMax=3",
			"-o", "BatchMode=yes",
			"-o", "StrictHostKeyChecking=yes",
			hostAlias,
		},
	}
}
