package voting

type (
	VotingScore int

	Vote struct {
		UserId string      `json:"user"`
		Score  VotingScore `json:"vote"`
	}
)

const (
	ScoreUp      VotingScore = 1
	ScoreDiscard VotingScore = 0
	ScoreDown    VotingScore = -1
)
