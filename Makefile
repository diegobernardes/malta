configure:
	@git config pull.rebase true
	@git config remote.origin.prune true
	@git config branch.master.mergeoptions "--ff-only"