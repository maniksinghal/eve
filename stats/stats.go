package stats

type Stat_data struct {
	Timestamp    string
	Id           int
	Query        string
	Category     string
	NumResponses int
	FullResponse string
}

type Stats_handle interface {
	Updatestat(query string, category string, num_responses int,
		full_response string) error
	Initialize() error
	GetResponseById(Id int) (string, error)
	GetLastNstats(last_n int) ([]Stat_data, error)
}

var Stats Stats_handle
