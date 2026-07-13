package local

// Prepare creates the fixed local layout and installs shared per-user assets.
func Prepare() error {
	layout, err := ResolveLayout()
	if err != nil {
		return err
	}
	if err := initializeDirs(layout); err != nil {
		return err
	}
	if err := installShellIntegration(layout); err != nil {
		return err
	}
	return installLicenses(layout)
}
