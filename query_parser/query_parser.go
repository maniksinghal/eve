package queryparser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/maniksinghal/eve/stats"
	schema_db "github.com/maniksinghal/eve/timing-db"
)

type query_handler_t func(query string, matches []string) (output string)

var query_map map[string]query_handler_t

func hello_world_handler(query string, matches []string) string {
	return ""
}

func Initialize() {
	query_map = make(map[string]query_handler_t)
	query_map["hello_world"] = hello_world_handler
}

func check_pid_db_query(query string, db schema_db.Schema_db) (result string,
	matched bool) {
	var response string
	var keywords = strings.Split(query, " ")
	responses, matched_families, matched_pids := schema_db.Query_database(keywords, db)

	matched = true
	for _, resp := range responses {
		this_resp := fmt.Sprintf("The %s family %s card uses %s=%s",
			resp.Family, resp.Pid, resp.Property, resp.Value)
		if len(resp.Port_range) > 0 {
			this_resp = this_resp + fmt.Sprintf(" on ports %s", resp.Port_range)
		}
		if len(resp.Lane_speeds) > 0 {
			this_resp = this_resp + fmt.Sprintf(" with speeds %s", resp.Lane_speeds)
		}

		response = response + this_resp + "\n"
	}

	if len(responses) == 0 {
		if strings.Contains(query, "everything") {
			if len(matched_families) == 1 {
				response = schema_db.Get_family_info(db, matched_families[0])
			} else if len(matched_pids) == 1 {
				response = schema_db.Get_pid_info(db, matched_pids[0])
			} else {
				matched = false
			}
		} else {
			matched = false
		}
	}

	if matched {
		stats.Stats.Updatestat(query, "PID_INFO", len(responses), response)
	}
	return response, matched
}

func Parse_query(query string, db schema_db.Schema_db) (result string) {

	no_match_reply := "Sorry, please try a simpler query\nType help for examples"

	response, matched := check_pid_db_query(query, db)
	if matched {
		return response
	}

	for regex, func_obj := range query_map {
		regex_obj := regexp.MustCompile(regex)
		matches := regex_obj.FindAllString(query, -1)
		if matches != nil {
			result := func_obj(query, matches)
			return result
		}
	}

	/* No match */
	return no_match_reply
}
