package usecase

type Facade struct {
	Task TaskUsecase
}

func NewFacade(
	TaskUC TaskUsecase,
) *Facade {
	return &Facade{
		Task: TaskUC,
	}
}
