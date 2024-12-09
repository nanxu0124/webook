package logger

func Error(err error) Field {
	return Field{
		Key:   "error",
		Value: err,
	}
}
