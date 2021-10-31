package timing_db_schema

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

type T_lane_speed struct {
	Speeds     []string
	Properties map[string]interface{}
}

type T_port_range struct {
	Start       int
	Stop        int
	Properties  map[string]interface{}
	Lane_speeds []T_lane_speed
}
type T_pids struct {
	Properties  map[string]interface{}
	Port_ranges []T_port_range
}
type T_families struct {
	Properties map[string]interface{}
	Pids       []T_pids
}

type Schema_db interface {
	Parse_db(string) error
	Get_families() []T_families
}

var terms_map map[string][]string

func load_term_mappings() {
	fileBytes, err := ioutil.ReadFile("multi_meaning_terms.txt")
	if err != nil {
		panic(err)
	}

	terms_map = make(map[string][]string)

	multi_meaning_terms := strings.Split(string(fileBytes), "\n")
	for _, entry := range multi_meaning_terms {
		term_meanings := strings.Split(entry, ":")
		if len(term_meanings) != 2 {
			// Expected term : <comma-separated meanings>
			panic(fmt.Errorf("incorrect term-map: %s", entry))
		}

		term := strings.TrimSpace(strings.ToLower(term_meanings[0]))
		meanings := strings.Split(strings.ToLower(term_meanings[1]), ",")
		terms_map[term] = meanings
		fmt.Printf("Added term-map %s => %s\n", term, term_meanings[1])
	}
}

func Parse_database(db Schema_db, filepath string) {
	db.Parse_db(filepath)
	load_term_mappings()

	/*
	 * Fix optimize_families:
	 * Once a property is exported, delete from children
	 * But remember when property is queried for a specific child
	 */
	optimize_families(db.Get_families())
}

/*
 * Promote common properties of children to their parents
 */
func optimize_port_ranges(ranges []T_port_range) {

	for _, prange := range ranges {

		fmt.Printf("Going to optimize range %d-%d\n", prange.Start, prange.Stop)

		/*
		 * Now optimize this port-range
		 */
		var common_lane_props map[string]interface{}
		common_lane_props = nil
		for _, lane := range prange.Lane_speeds {
			if common_lane_props == nil {
				total_props := len(lane.Properties)
				common_lane_props = make(map[string]interface{}, total_props)
				for prop, value := range lane.Properties {
					common_lane_props[prop] = value
				}
				continue
			}

			var props_to_delete []string
			for prop, value := range common_lane_props {
				_, exists := lane.Properties[prop]
				if exists && value == lane.Properties[prop] {
					continue
				} else {
					props_to_delete = append(props_to_delete, prop)
				}
			}

			for _, prop := range props_to_delete {
				delete(common_lane_props, prop)
			}

			if len(common_lane_props) == 0 {
				break
			}
		}

		/*
		 * common_range_props have props which have same value
		 * across all PIDs
		 */
		for prop, value := range common_lane_props {
			prange.Properties[prop] = value
			fmt.Printf("Promoted propery %s=%s to port-range %d-%d\n", prop,
				value, prange.Start, prange.Stop)
		}
	}
}

/*
 * Promote common properties of children to their parents
 */
func optimize_pids(pids []T_pids) {

	for _, pid := range pids {

		fmt.Printf("Going to optimize %s\n", pid.Properties["name"])

		optimize_port_ranges(pid.Port_ranges)

		/*
		 * Now optimize this pid
		 */
		var common_range_props map[string]interface{}
		common_range_props = nil
		for _, prange := range pid.Port_ranges {
			if common_range_props == nil {
				total_props := len(prange.Properties)
				common_range_props = make(map[string]interface{}, total_props)
				for prop, value := range prange.Properties {
					common_range_props[prop] = value
				}
				continue
			}

			var props_to_delete []string
			for prop, value := range common_range_props {
				_, exists := prange.Properties[prop]
				if exists && value == prange.Properties[prop] {
					continue
				} else {
					props_to_delete = append(props_to_delete, prop)
				}
			}

			for _, prop := range props_to_delete {
				delete(common_range_props, prop)
			}

			if len(common_range_props) == 0 {
				break
			}
		}

		/*
		 * common_range_props have props which have same value
		 * across all PIDs
		 */
		for prop, value := range common_range_props {
			pid.Properties[prop] = value
			fmt.Printf("Promoted propery %s=%s to pid %s\n", prop,
				value, pid.Properties["name"])
		}
	}
}

/*
 * Promote common properties of children to their parents
 */
func optimize_families(families []T_families) {

	for _, family := range families {

		optimize_pids(family.Pids)

		/*
		 * Now optimize this family
		 */
		var common_pid_props map[string]interface{}
		common_pid_props = nil
		for _, pid := range family.Pids {
			if common_pid_props == nil {
				total_props := len(pid.Properties)
				common_pid_props = make(map[string]interface{}, total_props)
				for prop, value := range pid.Properties {
					common_pid_props[prop] = value
				}

				continue
			}

			var props_to_delete []string
			for prop, value := range common_pid_props {
				_, exists := pid.Properties[prop]
				if exists && value == pid.Properties[prop] {
					continue
				} else {
					props_to_delete = append(props_to_delete, prop)
				}
			}

			for _, prop := range props_to_delete {
				delete(common_pid_props, prop)
			}

			if len(common_pid_props) == 0 {
				break
			}
		}

		/*
		 * common_pid_props have props which have same value
		 * across all PIDs
		 */
		for prop, value := range common_pid_props {
			family.Properties[prop] = value
			fmt.Printf("Promoted propery %s=%s to family %s\n", prop,
				value, family.Properties["name"])
		}
	}
}

type Query_response struct {
	Family           string
	Pid              string
	Port_range       string
	Lane_speeds      string
	Property         string
	Value            string
	Property_matched bool
	Value_matched    bool
}

/*
 * Maintain whether property-name matched or the value matched in the
 * query
 */
type matched_properties struct {
	property       string
	property_match bool
	value_match    bool
}

func does_term_match(keyword string, term string) (bool, string) {

	/* First check if the term itself matches */
	if strings.EqualFold(keyword, term) {
		return true, term
	}

	/* Check if any of the other synonyms of the term match */
	term = strings.ToLower(term)
	if meanings, exist := terms_map[term]; exist {
		for _, meaning := range meanings {
			meaning = strings.TrimSpace(meaning)

			if strings.EqualFold(keyword, meaning) {
				return true, meaning
			}
		}
	}

	return false, ""

}
func check_match(family string, pid string, query []string,
	properties map[string]interface{}) (matched_props []matched_properties, matched bool) {

	var prop_name_matched bool = false
	var value_matched bool = false

	for prop, value := range properties {
		prop = strings.TrimSpace(prop)
		// Values can be organized as <value> | Comments
		value_extract := strings.Split(value.(string), "|")[0]
		value_array := strings.Split(value_extract, ",")
		prop_name_matched = false
		value_matched = false

		prop_to_match := strings.Replace(prop, " ", "-", -1)

		for _, keyword := range query {
			keyword = strings.TrimSpace(keyword)
			if len(keyword) == 0 {
				continue
			}

			for _, sub_value := range value_array {
				sub_value = strings.TrimSpace(sub_value)

				/*
				 * For multi-word value or property-name, hyphenate them
				 */
				sub_value = strings.Replace(sub_value, " ", "-", -1)

				matched, with_term := does_term_match(keyword, sub_value)
				if matched {
					fmt.Printf("Matched value %s with %s=%s(%s) in %s/%s\n",
						keyword, prop, sub_value, with_term, family, pid)
					value_matched = true
					break
				}
			}

			matched, with_term := does_term_match(keyword, prop_to_match)
			if matched {
				fmt.Printf("Matched property %s with %s(%s)=%s in %s/%s\n",
					keyword, prop, with_term, value, family, pid)
				prop_name_matched = true
			}
		}

		if prop_name_matched || value_matched {
			matched_prop := new(matched_properties)
			matched_prop.property = prop
			matched_prop.property_match = prop_name_matched
			matched_prop.value_match = value_matched
			matched_props = append(matched_props, *matched_prop)
		}
	}

	if len(matched_props) > 0 {
		return matched_props, true
	} else {
		return matched_props, false
	}
}

func port_range_to_string(port_range *T_port_range) string {
	var ids []string
	ids = append(ids, strconv.Itoa((*port_range).Start))
	ids = append(ids, strconv.Itoa((*port_range).Stop))
	return strings.Join(ids, ",")

}

func query_lanes(family string, pid string, prange string,
	lanes []T_lane_speed, query []string) []Query_response {
	var responses []Query_response

	for _, lane := range lanes {
		props, matched := check_match(family, pid, query, lane.Properties)
		if matched {
			for _, prop := range props {
				var response = new(Query_response)
				response.Family = family
				response.Pid = pid
				response.Property = prop.property
				response.Port_range = prange
				response.Lane_speeds = strings.Join(lane.Speeds, ",")
				response.Value = lane.Properties[prop.property].(string)
				response.Property_matched = prop.property_match
				response.Value_matched = prop.value_match
				responses = append(responses, *response)
			}
		}
	}

	return responses
}

func query_ranges(family string, pid string, query []string, ranges []T_port_range) []Query_response {
	var responses []Query_response
	var lane_responses []Query_response

	for _, prange := range ranges {
		props, matched := check_match(family, pid, query, prange.Properties)
		if matched {
			for _, prop := range props {
				var response = new(Query_response)
				response.Family = family
				response.Pid = pid
				response.Property = prop.property
				response.Port_range = port_range_to_string(&prange)
				response.Value = prange.Properties[prop.property].(string)
				response.Property_matched = prop.property_match
				response.Value_matched = prop.value_match
				responses = append(responses, *response)
			}
		}

		lane_responses = query_lanes(family, pid, port_range_to_string(&prange),
			prange.Lane_speeds, query)

		// append slice
		responses = append(responses, lane_responses...)
	}

	return responses
}

func query_pids(family string, pids []T_pids, query []string) ([]Query_response,
	[]string) {
	var matched_pids []string
	var responses []Query_response

	for _, pid := range pids {
		pid_name := pid.Properties["name"].(string)
		props, matched := check_match(family, pid_name, query,
			pid.Properties)
		//fmt.Printf("Checking keywords:%s for pid:%s\n", strings.Join(query, ","), pid_name)
		if matched {
			for _, prop := range props {
				if strings.EqualFold(prop.property, "name") || strings.EqualFold(prop.property, "Internal name") {
					matched_pids = append(matched_pids, pid_name)
				} else {
					var response = new(Query_response)
					response.Family = family
					response.Pid = pid_name
					response.Property = prop.property
					response.Property_matched = prop.property_match
					response.Value_matched = prop.value_match
					response.Value = pid.Properties[prop.property].(string)
					responses = append(responses, *response)
				}
			}
		}

		range_responses := query_ranges(family, pid_name, query, pid.Port_ranges)
		responses = append(responses, range_responses...)

	}

	return responses, matched_pids
}

func prefer_both_prop_and_value_matches(responses []Query_response) []Query_response {
	var filtered_responses []Query_response

	for i := range responses {
		if responses[i].Property_matched && responses[i].Value_matched {
			filtered_responses = append(filtered_responses, responses[i])
		}
	}

	if len(filtered_responses) > 0 {
		return filtered_responses
	} else {
		return responses
	}
}

/*
 * If a term matches both a property-name and a value-name, then
 * prefer results with property-name.
 */
func prefer_property_over_value(responses []Query_response) []Query_response {
	var filtered_responses []Query_response

	var property_matches []Query_response
	for i := range responses {
		if responses[i].Property_matched && !responses[i].Value_matched {
			property_matches = append(property_matches, responses[i])
		}
	}

	if len(property_matches) == 0 {
		/* Only value matches */
		return responses
	}

	for i := range responses {
		include := true
		if responses[i].Value_matched && !responses[i].Property_matched {
			for j := range property_matches {
				for _, res := range strings.Split(responses[i].Value, ",") {
					if strings.EqualFold(property_matches[j].Property, res) {
						fmt.Printf("Excluding result for value %s=%s/%s/%s as it also matches the property %s=%s\n",
							responses[i].Property, res, responses[i].Pid, responses[i].Family,
							property_matches[j].Property, property_matches[j].Value)
						include = false
						break
					}
				}

				if !include {
					break
				}
			}
		}

		if include {
			filtered_responses = append(filtered_responses, responses[i])
		}
	}

	return filtered_responses
}

/*
 * If some stronger matches are present, then don't print
 * generic matches (like family=fretta)
 */
func remove_generic_matches(responses []Query_response) []Query_response {

	var filtered_responses []Query_response
	for i := range responses {
		if !strings.EqualFold("family", responses[i].Property) {
			filtered_responses = append(filtered_responses, responses[i])
		} else {
			fmt.Printf("Found generic result %s/%s. May remove below\n",
				responses[i].Pid, responses[i].Family)
		}
	}

	if len(filtered_responses) > 0 {
		// Found some stronger matches
		return filtered_responses
	} else {
		return responses
	}
}

/*
 * As properties are promoted from children to parents, both start showing
 * results. Remove duplicate results from children and show only at parent
 * level
 */
func remove_duplicate_matches(responses []Query_response) []Query_response {

	var filtered_responses []Query_response

	/* First find results only with port-range properties */
	var port_range_props []Query_response
	for i, _ := range responses {
		if len(responses[i].Lane_speeds) == 0 &&
			len(responses[i].Port_range) > 0 {
			port_range_props = append(port_range_props, responses[i])
		}
	}
	fmt.Printf("Built port-range properties of length %d\n", len(port_range_props))

	/* Now remove results which match lane-range properties */
	for i, _ := range responses {
		include := true
		if len(responses[i].Lane_speeds) > 0 {
			for j, _ := range port_range_props {
				if responses[i].Port_range == port_range_props[j].Port_range &&
					responses[i].Property == port_range_props[j].Property &&
					responses[i].Pid == port_range_props[j].Pid &&
					responses[i].Family == port_range_props[j].Family {
					include = false
					fmt.Printf("Excluding %s/%s/%s/%s. Already included in port-range\n",
						responses[i].Property, responses[i].Lane_speeds,
						responses[i].Port_range, responses[i].Pid)
					break
				}
			}

		}

		if include {
			filtered_responses = append(filtered_responses, responses[i])
		}
	}

	responses = filtered_responses

	/* Find results only with PID properties */
	var pid_responses []Query_response
	for i, _ := range responses {
		if len(responses[i].Port_range) == 0 &&
			len(responses[i].Pid) > 0 {
			pid_responses = append(pid_responses, responses[i])
		}
	}

	/* Now remove results which match port-range properties */
	filtered_responses = nil
	for i, _ := range responses {
		include := true
		if len(responses[i].Port_range) > 0 {
			for j, _ := range pid_responses {
				if responses[i].Property == pid_responses[j].Property &&
					responses[i].Pid == pid_responses[j].Pid &&
					responses[i].Family == pid_responses[j].Family {
					include = false
					fmt.Printf("Excluding %s/%s/%s. Already included in Pid\n",
						responses[i].Property, responses[i].Port_range,
						responses[i].Pid)
					break
				}
			}
		}

		if include {
			filtered_responses = append(filtered_responses, responses[i])
		}
	}

	responses = filtered_responses

	/* Find results only with Family properties */
	var family_responses []Query_response
	for i, _ := range responses {
		if len(responses[i].Pid) == 0 &&
			len(responses[i].Family) > 0 {
			pid_responses = append(family_responses, responses[i])
		}
	}

	/* Now remove results which match PID properties */
	filtered_responses = nil
	for i, _ := range responses {
		include := true
		if len(responses[i].Pid) > 0 {
			for j, _ := range family_responses {
				if responses[i].Property == pid_responses[j].Property &&
					responses[i].Family == pid_responses[j].Family {
					include = false
					fmt.Printf("Excluding %s/%s/%s. Already included in family\n",
						responses[i].Property, responses[i].Pid,
						responses[i].Family)
					break
				}
			}
		}

		if include {
			filtered_responses = append(filtered_responses, responses[i])
		}
	}

	return filtered_responses
}

/*
 * If we have a value1 matching the query with prop1=value1, then remove
 * all other matches where prop1=value2, prop1=value2 and so on.
 */
func filter_unique_value_matches(responses []Query_response) []Query_response {
	var value_matches []Query_response
	var filtered_responses []Query_response

	/* First collect all value matches */
	for i := range responses {
		//fmt.Printf("filter_unique_value_matches: Scanning %s=>%s\n",
		//	responses[i].Property, responses[i].Value)
		if responses[i].Value_matched {
			value_matches = append(value_matches, responses[i])
		}
	}

	/* Now scan for all responses and find properties that matched but
	 * with different value
	 */
	for i := range responses {
		var include bool = true
		for v := range value_matches {
			if responses[i].Property == value_matches[v].Property &&
				!responses[i].Value_matched {
				include = false
				break
			}
		}

		if include {
			filtered_responses = append(filtered_responses, responses[i])
		}
	}
	return filtered_responses
}

func Query_database(query []string, db Schema_db) (responses []Query_response,
	matched_families []string, matched_pids []string) {

	for _, family := range db.Get_families() {
		family_name := family.Properties["name"].(string)
		props, matched := check_match(family_name, "all", query,
			family.Properties)
		if matched {
			for _, prop := range props {
				if prop.property == "name" {
					matched_families = append(matched_families, family_name)
				} else {
					var response = new(Query_response)
					response.Family = family_name
					response.Pid = ""
					response.Property = prop.property
					response.Property_matched = prop.property_match
					response.Value_matched = prop.value_match
					response.Value = family.Properties[prop.property].(string)
					responses = append(responses, *response)
				}
			}
		}

		pid_responses, matched_pids_in_family :=
			query_pids(family_name, family.Pids, query)
		responses = append(responses, pid_responses...)

		matched_pids = append(matched_pids, matched_pids_in_family...)
	}
	//fmt.Printf("Got matched pids: %s\n", strings.Join(matched_pids, ","))

	var filtered_responses []Query_response
	for _, response := range responses {
		if len(matched_pids) > 0 {
			// Some pid matched
			for _, pid := range matched_pids {
				if response.Pid == pid {
					filtered_responses = append(filtered_responses, response)
					break
				}
			}
		} else if len(matched_families) > 0 {
			for _, family := range matched_families {
				if response.Family == family {
					filtered_responses = append(filtered_responses, response)
				}
			}
		} else {
			// If no family or PID match, allow all matched keywords
			filtered_responses = append(filtered_responses, response)
		}
	}

	/*
	 * If a query matched both the property-name and one of its specific values,
	 * then filter out all entries where same property has a different value
	 */
	filtered_responses = filter_unique_value_matches(filtered_responses)

	/*
	 * If a query matched both property-name and one of its values, then
	 * prefer only those both match responses
	 */
	filtered_responses = prefer_both_prop_and_value_matches(filtered_responses)

	/*
	 * Remove duplicate matches (properties matching both parent and children)
	 */
	filtered_responses = remove_duplicate_matches(filtered_responses)

	/*
	 * When stronger results are available, remove generic results
	 * like family=fretta
	 */
	filtered_responses = remove_generic_matches(filtered_responses)

	/*
	 * If a term matches both a property-name and some value-name of a different
	 * property as well then prefer property-name (useful especially for
	 * keywords like PHY/NPU which can be both a property and a value)
	 */
	filtered_responses = prefer_property_over_value(filtered_responses)

	return filtered_responses, matched_families, matched_pids
}

func Dump_lane_speed(lane_ptr *T_lane_speed) (response string) {

	output := "'" + strings.Join((*lane_ptr).Speeds, `','`) + `'`
	response = fmt.Sprintf("-     Printing lane_speeds %s\n", output)

	for prop, value := range (*lane_ptr).Properties {
		response = response + fmt.Sprintf("-       %s: %s\n", prop, value)
	}
	return response
}

func Dump_port_range(range_ptr *T_port_range) (response string) {

	response = fmt.Sprintf("-   Printing range %d-%d\n",
		(*range_ptr).Start, (*range_ptr).Stop)
	for prop, value := range (*range_ptr).Properties {
		response = response + fmt.Sprintf("-     %s: %s\n", prop, value)
	}

	// Now print port-ranges
	for _, lane_speed := range (*range_ptr).Lane_speeds {
		response = response + Dump_lane_speed(&lane_speed)
	}
	return response
}

func Dump_pid(pid_ptr *T_pids) (response string) {

	response = fmt.Sprintf("- Printing PID %s\n", (*pid_ptr).Properties["name"])
	for prop, value := range (*pid_ptr).Properties {
		response = response + fmt.Sprintf("-   %s: %s\n", prop, value)
	}

	// Now print port-ranges
	for _, port_range := range (*pid_ptr).Port_ranges {
		response = response + Dump_port_range(&port_range)
	}

	return response
}

func Dump_family(family *T_families) (response string) {
	response = fmt.Sprintf("Printing family %s\n", (*family).Properties["name"])
	for prop, value := range (*family).Properties {
		response = response + fmt.Sprintf("- %s: %s\n", prop, value)
	}

	// Now print pids
	for _, pid := range (*family).Pids {
		response = response + Dump_pid(&pid)
	}

	return response
}

func Dump_db(db Schema_db) {
	var response string
	families := db.Get_families()
	for _, family := range families {
		response = Dump_family(&family)
		fmt.Print(response)
	}
}

func Get_family_info(db Schema_db, family_name string) (response string) {
	for _, family := range db.Get_families() {
		if strings.EqualFold(family.Properties["name"].(string), family_name) {
			response = Dump_family(&family)
		}
	}
	return response
}

func Get_pid_info(db Schema_db, pid_name string) (response string) {
	for _, family := range db.Get_families() {
		for _, pid := range family.Pids {
			if strings.EqualFold(pid_name, pid.Properties["name"].(string)) ||
				strings.EqualFold(pid_name, pid.Properties["Internal name"].(string)) {
				response = Dump_pid(&pid)
			}
		}
	}
	return response
}
