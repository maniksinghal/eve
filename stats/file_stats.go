package stats

import (
	"log"
	"os"
)

type File_stats struct {
	stats_fd *log.Logger
}

func (handle *File_stats) Updatestat(query string, requestor string, category string, num_responses int,
	full_response string) error {
	handle.stats_fd.Printf("From:%s, Q:%s, Cat:%s, Results:%d\n", requestor,
		query, category, num_responses)
	return nil
}

func (handle *File_stats) Initialize(user string, password string, host string, port int) error {

	f, err := os.OpenFile("eve.stats", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	//f.Truncate(0)
	handle.stats_fd = log.New(f, "STAT: ", log.Ldate|log.Ltime)

	Stats = handle
	return nil
}

func (handle *File_stats) GetResponseById(Id int) (string, error) {
	return "", nil
}

func (handle *File_stats) GetLastNstats(last_n int) ([]Stat_data, error) {
	return nil, nil
}
