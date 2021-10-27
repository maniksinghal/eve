package timing_db_schema

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Json_db struct {
	families []T_families
}

func (db *Json_db) fill_port_range(port_range string,
	port_range_ptr *T_port_range,
	port_range_data interface{}) {

	(*port_range_ptr).Properties = port_range_data.(map[string]interface{})

	// @todo: validate before assuming X-Y format
	start_stop := strings.Split(port_range, "-")
	(*port_range_ptr).Start, _ = strconv.Atoi(start_stop[0])
	(*port_range_ptr).Stop, _ = strconv.Atoi(start_stop[1])

	_, exists := (*port_range_ptr).Properties["lane-speeds"]
	if !exists {
		return
	}

	var lane_speeds map[string]interface{} = (*port_range_ptr).Properties["lane-speeds"].(map[string]interface{})
	total_lane_speeds := len(lane_speeds)
	(*port_range_ptr).Lane_speeds = make([]T_lane_speed, total_lane_speeds)

	next_lane_speed := 0
	for lane_speed, lane_speed_data := range lane_speeds {
		var lane_speed_ptr *T_lane_speed = &((*port_range_ptr).Lane_speeds[next_lane_speed])

		// Todo: derive possible lane speeds from lane_speed string
		(*lane_speed_ptr).Speeds = strings.Split(lane_speed, ",")

		(*lane_speed_ptr).Properties = lane_speed_data.(map[string]interface{})
		//(*lane_speed_ptr).Properties["name"] = lane_speed
		next_lane_speed = next_lane_speed + 1
	}

	// Remove lane-speeds from properties. It is already filled
	// in the lane_speeds structure
	delete((*port_range_ptr).Properties, "lane-speeds")
}

func (db *Json_db) fill_pid(pid string, pid_ptr *T_pids,
	pid_data interface{}) {
	(*pid_ptr).Properties = pid_data.(map[string]interface{})
	(*pid_ptr).Properties["name"] = pid

	_, exists := (*pid_ptr).Properties["port-ranges"]
	if !exists {
		return
	}

	var port_ranges map[string]interface{} = (*pid_ptr).Properties["port-ranges"].(map[string]interface{})
	total_port_ranges := len(port_ranges)
	(*pid_ptr).Port_ranges = make([]T_port_range, total_port_ranges)

	next_port_range := 0
	for port_range, port_range_data := range port_ranges {
		db.fill_port_range(port_range, &((*pid_ptr).Port_ranges[next_port_range]),
			port_range_data)
		next_port_range = next_port_range + 1
	}

	// Remove port-ranges from properties. It is already filled
	// in the port_ranges structure
	delete((*pid_ptr).Properties, "port-ranges")

}

func (db *Json_db) fill_family(family_index int,
	family string,
	family_data interface{}) {

	var family_ptr *T_families = &db.families[family_index]
	(*family_ptr).Properties = family_data.(map[string]interface{})
	(*family_ptr).Properties["name"] = family

	var pids map[string]interface{} = (*family_ptr).Properties["pids"].(map[string]interface{})
	total_pids := len(pids)
	(*family_ptr).Pids = make([]T_pids, total_pids)

	next_pid := 0
	for pid, pid_data := range pids {
		db.fill_pid(pid, &((*family_ptr).Pids[next_pid]), pid_data)
		next_pid = next_pid + 1
	}

	// Remove pids from properties. It is already filled
	// in the pids structure
	delete((*family_ptr).Properties, "pids")
}

/*
 * Parse a json file database
 */
func (db *Json_db) Parse_db(filepath string) error {
	jsonBytes, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(jsonBytes), &result)

	var families map[string]interface{} = result["root"].(map[string]interface{})

	db.families = make([]T_families, len(families))

	next_family := 0
	for family, family_data := range families {
		fmt.Println("Family is: " + family)
		db.fill_family(next_family, family, family_data)
		next_family = next_family + 1
	}

	return nil
}

func (db Json_db) Get_families() []T_families {
	return db.families
}
