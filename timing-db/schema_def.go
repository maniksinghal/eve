package timing_db_schema

import (
	"fmt"
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

func Parse_database(db Schema_db, filepath string) {
	db.Parse_db(filepath)

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
	Family      string
	Pid         string
	Port_range  string
	Lane_speeds string
	Property    string
	Value       string
}

func check_match(family string, pid string, query []string,
	properties map[string]interface{}) (props []string, matched bool) {

	var matched_props []string
	for _, keyword := range query {
		for prop, value := range properties {
			value_array := strings.Split(value.(string), ",")
			for _, sub_value := range value_array {
				if keyword == sub_value || keyword == prop {
					fmt.Printf("Matched keyword %s with %s=%s in %s/%s\n",
						keyword, prop, value, family, pid)
					matched_props = append(matched_props, prop)
				}
			}
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
				response.Property = prop
				response.Port_range = prange
				response.Lane_speeds = strings.Join(lane.Speeds, ",")
				response.Value = lane.Properties[prop].(string)
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
				response.Property = prop
				response.Port_range = port_range_to_string(&prange)
				response.Value = prange.Properties[prop].(string)
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
		fmt.Printf("Checking keywords:%s for pid:%s\n", strings.Join(query, ","), pid_name)
		if matched {
			for _, prop := range props {
				if prop == "name" || prop == "nickname" {
					matched_pids = append(matched_pids, pid_name)
				} else {
					var response = new(Query_response)
					response.Family = family
					response.Pid = pid_name
					response.Property = prop
					response.Value = pid.Properties[prop].(string)
					responses = append(responses, *response)
				}
			}
		}

		range_responses := query_ranges(family, pid_name, query, pid.Port_ranges)
		responses = append(responses, range_responses...)

	}

	return responses, matched_pids
}

func Query_database(query []string, db Schema_db) (responses []Query_response,
	matched_families []string, matched_pids []string) {

	for _, family := range db.Get_families() {
		family_name := family.Properties["name"].(string)
		props, matched := check_match(family_name, "all", query,
			family.Properties)
		if matched {
			for _, prop := range props {
				if prop == "name" {
					matched_families = append(matched_families, family_name)
				} else {
					var response = new(Query_response)
					response.Family = family_name
					response.Pid = ""
					response.Property = prop
					response.Value = family.Properties[prop].(string)
					responses = append(responses, *response)
				}
			}
		}

		pid_responses, matched_pids_in_family :=
			query_pids(family_name, family.Pids, query)
		responses = append(responses, pid_responses...)

		matched_pids = append(matched_pids, matched_pids_in_family...)
	}
	fmt.Printf("Got matched pids: %s\n", strings.Join(matched_pids, ","))

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
