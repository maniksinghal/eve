package queryparser

import (
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	"github.com/maniksinghal/eve/stats"
	schema_db "github.com/maniksinghal/eve/timing-db"
)

type query_handler_t func(db *schema_db.Schema_db, query string, matches []string) (output string)

var query_map map[string]query_handler_t
var multi_word_terms []string

var Query_added_to_room string = "eveaddedtoroom"

func refresh_timing_db(db *schema_db.Schema_db, query string, matches []string) string {
	timing_db_file := (*db).Get_db_source_file()
	schema_db.Parse_database(*db, timing_db_file)
	return "Timing DB parsing complete"
}

func added_to_room_handler(db *schema_db.Schema_db, query string, matches []string) string {
	response := "Hi, I am Eve - a timing chatbot\n"
	response += "Type **help** to check what all I can do"
	return response
}

func show_usage_handler(db *schema_db.Schema_db, query string, matches []string) string {
	response := "I can answer simple queries like:\n\n"
	response += "- *Which card is NCS-55A1-24Q6H-S*\n"
	response += "- OR *Which all types of ports are there in eyrie*\n"
	response += "\n"
	response += "or queries on **Timing capabilities** like:\n"
	response += "- *Which PHY is used on Everglades*\n"
	response += "- *Which all Felidae cards use MetaDX phy*\n"
	response += "- *Where is timestamping done on Tortin*\n"
	response += "- *Where is Synce clock recovered on Acadia*\n"
	response += "- *Does shadow-tower support GNSS*\n"
	response += "- *Does Denali have bits port*\n"
	response += "- *Does peyto support class-C timing*\n"
	response += "\n"
	response += "or queries on **Supported Features** like:\n"
	response += "- *Is virtual PTP port supported on Peyto*\n"
	response += "- *which all cards support eSynce*\n"
	response += "- *Does old castle support timing on breakout ports*\n"

	return response
}

func show_test_commands(db *schema_db.Schema_db, query string, matches []string) string {
	var response string
	for regex := range query_map {
		response += regex
		response += "\n"
	}

	return response
}

func build_multi_word_terms(db *schema_db.Schema_db, query string, matches []string) string {
	fileBytes, err := ioutil.ReadFile("multi_word_terms.txt")
	if err != nil {
		panic(err)
	}

	multi_word_terms = strings.Split(string(fileBytes), "\n")
	count := 0
	for i := range multi_word_terms {
		log.Printf("Combining multi-word term %s\n", multi_word_terms[i])
		count += 1
	}

	return fmt.Sprintf("Parsed %d terms", count)
}

func Initialize() {
	query_map = make(map[string]query_handler_t)

	/*
	 * ADD ALL REGULAR EXPRESSIONS IN LOWER CASE ONLY
	 */
	query_map["test_bot.*refresh\\s+timing_db"] = refresh_timing_db
	query_map["test_bot.*refresh\\s+multi.word"] = build_multi_word_terms
	query_map[Query_added_to_room] = added_to_room_handler
	query_map["test_bot\\s+list"] = show_test_commands
	query_map["help\\s*$"] = show_usage_handler
	build_multi_word_terms(nil, "", nil) // Arguments not used by function
}

func pre_process_query(query string) (keywords []string) {

	//Lowercase the query
	query = strings.ToLower(query)

	//Remove punctuations sticking to the words
	var punctuation_list []string = []string{",", "?"}
	for _, punc := range punctuation_list {
		query = strings.Replace(query, punc, " ", -1)
	}
	log.Printf("Query after removing punctuations: %s\n", query)

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
			log.Printf("Replaced %s with %s\n", multi_word_string, hyphenated_string)

		}
	}

	/*
	 * Map variations to a common name
	 */

	return strings.Split(query, " ")
}

func check_pid_db_query(query string, sender string, db *schema_db.Schema_db) (result string,
	matched bool) {
	var response string
	var keywords = pre_process_query(query)
	responses, matched_families, matched_pids := schema_db.Query_database(keywords, db)

	too_many_responses := false
	if len(responses) > 50 {
		responses = responses[:50]
		too_many_responses = true
	}

	matched = true
	for _, resp := range responses {

		// Response value can be in <value> | <Comment> form
		value_comment := strings.Split(resp.Value, "|")
		value := strings.TrimSpace(value_comment[0])
		this_resp := fmt.Sprintf("The %s family card %s (%s) has **%s** = **%s**",
			resp.Family, resp.Pid, resp.Nickname, resp.Property, value)
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
			} else if len(matched_pids) > 0 {
				response = schema_db.Get_pid_info(db, matched_pids[0])
			} else {
				matched = false
			}
		} else if len(matched_pids) > 0 {
			response = ""
			for iter := range matched_pids {
				response += schema_db.Get_pid_summary(db, matched_pids[iter])
				response += "\n"
			}
		} else {
			matched = false
		}
	}

	if too_many_responses {
		response += "... Too many entries [truncated]. Please try a more specific query\n"
	}

	if matched {
		stats.Stats.Updatestat(query, sender, "PID_INFO", len(responses), response)
	}
	return response, matched
}

func Parse_query(query string, sender string, db *schema_db.Schema_db) (result string) {

	// Empty string check
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return ""
	}

	no_match_reply := "Sorry, please try to rephrase or simplify the query\n"
	no_match_reply += "Type **help** for examples"

	response, matched := check_pid_db_query(query, sender, db)
	if matched {
		return response
	}

	for regex, func_obj := range query_map {
		regex_obj := regexp.MustCompile(regex)
		matches := regex_obj.FindAllString(strings.ToLower(query), -1)
		if matches != nil {
			result := func_obj(db, query, matches)
			stats.Stats.Updatestat(query, sender, "QUERY_MAP", 1, result)
			return result
		}
	}

	/* No match */
	stats.Stats.Updatestat(query, sender, "NO_MATCH", 0, no_match_reply)
	return no_match_reply
}
