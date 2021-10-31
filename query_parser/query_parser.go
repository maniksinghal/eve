package queryparser

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/maniksinghal/eve/stats"
	schema_db "github.com/maniksinghal/eve/timing-db"
)

type query_handler_t func(query string, matches []string) (output string)

var query_map map[string]query_handler_t
var multi_word_terms []string

func hello_world_handler(query string, matches []string) string {
	return ""
}

func Initialize() {
	query_map = make(map[string]query_handler_t)
	query_map["regular-expression"] = hello_world_handler

	fileBytes, err := ioutil.ReadFile("multi_word_terms.txt")
	if err != nil {
		panic(err)
	}

	multi_word_terms = strings.Split(string(fileBytes), "\n")
	for i := range multi_word_terms {
		fmt.Printf("Combining multi-word term %s\n", multi_word_terms[i])
	}
}

func pre_process_query(query string) (keywords []string) {

	//Lowercase the query
	query = strings.ToLower(query)

	// Remove accidental repeated whitespaces between words
	space := regexp.MustCompile(`\s+`)
	query = space.ReplaceAllString(query, " ")

	/*
	 * Combine multi-word phrases to a hyphen-separated words
	 */
	for i := range multi_word_terms {
		// Filter empty lines
		if len(multi_word_terms[i]) == 0 {
			continue
		}

		multi_word_string := strings.ToLower(multi_word_terms[i])
		if strings.Contains(query, multi_word_string) {
			hyphenated_string := strings.Replace(multi_word_string, " ", "-", -1)
			query = strings.Replace(query, multi_word_string, hyphenated_string, -1)
			fmt.Printf("Replaced %s with %s\n", multi_word_string, hyphenated_string)

		}
	}

	/*
	 * Map variations to a common name
	 */

	return strings.Split(query, " ")
}

func check_pid_db_query(query string, sender string, db schema_db.Schema_db) (result string,
	matched bool) {
	var response string
	var keywords = pre_process_query(query)
	responses, matched_families, matched_pids := schema_db.Query_database(keywords, db)

	matched = true
	for _, resp := range responses {

		// Response value can be in <value> | <Comment> form
		value_comment := strings.Split(resp.Value, "|")
		value := value_comment[0]
		this_resp := fmt.Sprintf("The %s family %s card uses %s=%s",
			resp.Family, resp.Pid, resp.Property, value)
		if len(value_comment) > 1 {
			// Comment also present
			this_resp += fmt.Sprintf(" (%s)", strings.TrimSpace(value_comment[1]))
		}
		if len(resp.Port_range) > 0 {
			this_resp = this_resp + fmt.Sprintf(" on ports %s", resp.Port_range)
		}
		if len(resp.Lane_speeds) > 0 {
			value_comment = strings.Split(resp.Lane_speeds, "|")
			value = value_comment[0]
			this_resp = this_resp + fmt.Sprintf(" with speeds %s", value)

			if len(value_comment) > 1 {
				// Comment also specified
				comment := strings.TrimSpace(value_comment[1])
				this_resp = this_resp + fmt.Sprintf(" (%s)", comment)
			}
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
		stats.Stats.Updatestat(query, sender, "PID_INFO", len(responses), response)
	}
	return response, matched
}

func Parse_query(query string, sender string, db schema_db.Schema_db) (result string) {

	// Empty string check
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return ""
	}

	no_match_reply := "Sorry, please try a simpler query\nType help for examples"

	response, matched := check_pid_db_query(query, sender, db)
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
