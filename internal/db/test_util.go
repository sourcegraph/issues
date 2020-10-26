package db

func MockEmailExistsErr() error {
	return errCannotCreateUser{errorCodeEmailExists}
}

func MockUsernameExistsErr() error {
	return errCannotCreateUser{errorCodeEmailExists}
}
