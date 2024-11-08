package worker

import "github.com/lmittmann/w3"

const (
	Approve      = "approve"
	Check        = "check"
	GiveTo       = "giveTo"
	MintTo       = "mintTo"
	NextTime     = "nextTime"
	Register     = "register"
	Transfer     = "transfer"
	TransferFrom = "transferFrom"
	Withdraw     = "withdraw"
	Deposit      = "deposit"
	Sweep        = "sweep"
)

var abi = map[string]*w3.Func{
	// ERC20
	Approve:  w3.MustNewFunc("approve(address, uint256)", "bool"),
	MintTo:   w3.MustNewFunc("mintTo(address, uint256)", "bool"),
	Transfer: w3.MustNewFunc("transfer(address,uint256)", "bool"),
	Sweep:    w3.MustNewFunc("sweep(address)", "uint256"),
	// GasFaucet
	Check:    w3.MustNewFunc("check(address)", "bool"),
	GiveTo:   w3.MustNewFunc("giveTo(address)", "uint256"),
	NextTime: w3.MustNewFunc("nextTime(address)", "uint256"),
	// CustodialRegistrationProxy
	Register: w3.MustNewFunc("register(address)", ""),
	// Pool
	Withdraw: w3.MustNewFunc("withdraw(address,address,uint256)", ""),
	Deposit:  w3.MustNewFunc("deposit(address,uint256)", ""),
}
