package timing_db_schema

import (
	"fmt"
	"log"
	"strconv"
	"strings"

    "github.com/360EntSecGroup-Skylar/excelize"
	//"github.com/xuri/excelize"
)

type Excel_db struct {
	file_handle *excelize.File
	file_name   string
	families    []T_families
}

type sheet_entry struct {
	keys       map[string]string
	properties map[string]string
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (db *Excel_db) Get_families() []T_families {
	return db.families
}

func (db *Excel_db) Get_db_source_file() string {
	return db.file_name
}

func (db *Excel_db) parse_sheet(sheet_name string, keys []string) (
	entries []sheet_entry, err error) {

	var col int = 0
	var row int = 0
	var num_rows int = 0
	var num_cols int = 0

	log.Printf("Parsing sheet %s\n", sheet_name)

	sheet_map, err := db.file_handle.GetRows(sheet_name)
	if err != nil {
		log.Println("Error opening sheet")
		return nil, fmt.Errorf("could not parse sheet %s", sheet_name)
	}

	num_rows = len(sheet_map)
	num_cols = len(sheet_map[0])

	/*
	 * Build a map of column-names
	 */
	headings_row := 0
	total_columns := 0
	col_names := make(map[int]string)
	col_indexes := make(map[string]int)
	col = 0
	for col < num_cols {
		col_name := sheet_map[headings_row][col]
		if col_name != "" {
			total_columns += 1
			col_names[col] = col_name
			col_indexes[col_name] = col
			log.Printf("Got column %s\n", col_name)
			col += 1
		} else {
			break
		}
	}
	log.Printf("Found total columns %d\n", total_columns)

	/*
	 * Verify the keys passed exist in the column names
	 */
	for _, key := range keys {
		_, ok := col_indexes[key]
		if !ok {
			return nil, fmt.Errorf("key %s not found in sheet %s",
				key, sheet_name)
		}
	}

	/*
	 * Now parse each entry
	 */
	row = 1
	var sheet_entries []sheet_entry
	for row < num_rows {
		col = 0
		entry := new(sheet_entry)
		entry.keys = make(map[string]string)
		entry.properties = make(map[string]string)

		/* Excel sheet rows start form 1 */
		entry.properties["_row_in_sheet"] = strconv.Itoa(row + 1)
		for col < num_cols {
			col_name := col_names[col]

			cell_value := ""
			if col < len(sheet_map[row]) {
				cell_value = sheet_map[row][col]
			}

			if stringInSlice(col_name, keys) {
				// @todo: validate that keys are unique across rows
				entry.keys[col_name] = cell_value
			} else {
				entry.properties[col_name] = cell_value
			}
			col += 1
		}

		// Ignroe entries in the sheet with AutoParse=ignore
		value, exists := entry.properties["AutoParse"]
		if !exists || value != "ignore" {
			sheet_entries = append(sheet_entries, *entry)
		} else {
			log.Printf("Ignoring entry at row %d in sheet %s\n",
				(row + 1), sheet_name)
		}
		row += 1
	}

	return sheet_entries, nil
}

func (db *Excel_db) Get_pid_from_pid_name(pid string) (pid_obj *T_pids) {
	for family_i := range db.families {
		for pid_i, pid_obj := range db.families[family_i].Pids {
			if pid_obj.Properties["name"] == pid {
				return &db.families[family_i].Pids[pid_i]
			}
		}
	}
	return nil
}

func (db *Excel_db) get_port_range_obj(pid string, port_range string) (port_range_obj *T_port_range) {
	for family_i := range db.families {
		for pid_i, pid_obj := range db.families[family_i].Pids {
			if pid_obj.Properties["name"] == pid {
				for port_i, port_range_obj := range db.families[family_i].Pids[pid_i].Port_ranges {
					start_stop := strings.Split(port_range, "-")
					start, _ := strconv.Atoi(start_stop[0])
					stop, _ := strconv.Atoi(start_stop[1])
					if start == port_range_obj.Start && stop == port_range_obj.Stop {
						return &db.families[family_i].Pids[pid_i].Port_ranges[port_i]
					}
				}
			}
		}
	}
	return nil
}

/*
 * Parse an xlsx file database
 */
func (db *Excel_db) Parse_db(filepath string) error {
	var families []T_families
	f, err := excelize.OpenFile(filepath)
	if err != nil {
		log.Println(err)
		return err
	}
	db.file_handle = f
	db.file_name = filepath

	sheet_list := f.GetSheetList()
	for _, sheet := range sheet_list {
		log.Println("Got sheet " + sheet)
	}

	keys := []string{"PID"}
	sheet_entries, err := db.parse_sheet("General", keys)
	if err != nil {
		f.Close()
		return err
	}

	for _, row := range sheet_entries {
		pid := row.keys["PID"]
		if pid == "" {
			return fmt.Errorf("could not find PID in the entry in sheet General, row:%s", row.properties["_row_in_sheet"])
		}

		// First scan the family
		family_name := row.properties["Family"]
		new_family := false
		var family_object *T_families = nil
		for iterator, family_obj := range families {
			family_obj_name := family_obj.Properties["name"].(string)
			if family_name == family_obj_name {
				family_object = &families[iterator]
				break
			}
		}

		if family_object == nil {
			// New family found in row, crate an object
			log.Printf("Created new family %s\n", family_name)
			family_object = new(T_families)
			family_object.Properties = make(map[string]interface{})
			family_object.Properties["name"] = family_name
			family_object.Pids = make([]T_pids, 0)
			new_family = true
		}

		// Insert PID into the family
		pid_obj := new(T_pids)
		pid_obj.Properties = make(map[string]interface{})
		pid_obj.Properties["name"] = pid
		pid_obj.Port_ranges = make([]T_port_range, 0)

		for key, value := range row.properties {
			pid_obj.Properties[key] = value
		}

		family_object.Pids = append(family_object.Pids, *pid_obj)

		log.Printf("Family %s, PID:%s, total pids:%d\n", family_name, pid,
			len(family_object.Pids))

		if new_family {
			families = append(families, *family_object)
		}
	}

	/*
	 * Now parse over port-ranges
	 */
	keys = []string{"PID", "Port range", "Port-type", "Speeds", "Internal name"}
	sheet_entries, err = db.parse_sheet("Port information", keys)
	if err != nil {
		f.Close()
		return err
	}

	db.families = families

	for _, row := range sheet_entries {
		var new_port_range bool = false
		pid := row.keys["PID"]
		port_range := row.keys["Port range"]
		speed := row.keys["Speeds"]
		port_type := row.keys["Port-type"]
		log.Printf("Parsing row %s, pid:%s\n", row.properties["_row_in_sheet"],
			pid)

		/*
		 * Validate that port-range should be in the order of M-N
		 */
		port_range_list := strings.Split(port_range, "-")
		if len(port_range_list) != 2 {
			return fmt.Errorf("invalid port range %s (not in format M-N) at row %s in Port information sheet",
				port_range, row.properties["_row_in_sheet"])
		}

		if pid == "" || port_range == "" || speed == "" {
			return fmt.Errorf("could not find PID/Port range or Speed in the entry in sheet Port information, row:%s", row.properties["_row_in_sheet"])
		}

		pid_obj := db.Get_pid_from_pid_name(pid)
		if pid_obj == nil {
			return fmt.Errorf("could not find PID %s in General sheet (referenced in row %s of Port information sheet",
				pid, row.properties["_row_in_sheet"])
		}

		pr_obj := db.get_port_range_obj(pid, port_range)
		if pr_obj == nil {
			pr_obj = new(T_port_range)
			new_port_range = true
			pr_obj.Start, _ = strconv.Atoi(strings.Split(port_range, "-")[0])
			pr_obj.Stop, _ = strconv.Atoi(strings.Split(port_range, "-")[1])
			pr_obj.Properties = make(map[string]interface{})
			pr_obj.Lane_speeds = make([]T_lane_speed, 0)
			log.Printf("Creating new port range for %s: %d-%d\n",
				pid, pr_obj.Start, pr_obj.Stop)

			pr_obj.Properties["Port-type"] = port_type
		} else {
			log.Printf("Found existing port-range for %s, while scanning %s\n",
				pid, row.properties["_row_in_sheet"])
		}

		lane_speed_obj := new(T_lane_speed)
		lane_speed_obj.Speeds = strings.Split(speed, ",")
		lane_speed_obj.Properties = make(map[string]interface{})

		for prop, value := range row.properties {
			lane_speed_obj.Properties[prop] = value
		}

		pr_obj.Lane_speeds = append(pr_obj.Lane_speeds, *lane_speed_obj)

		if new_port_range {
			pid_obj.Port_ranges = append(pid_obj.Port_ranges, *pr_obj)
		}
	}

	/*
	 * Now parse the Features sheet
	 */
	keys = []string{"PID", "Internal name"}
	sheet_entries, err = db.parse_sheet("Features", keys)
	if err != nil {
		log.Println(err.Error())
		f.Close()
		return err
	}

	for _, row := range sheet_entries {
		pid := row.keys["PID"]
		log.Printf("Scanning features sheet for PID %s\n", pid)

		if pid == "" {
			return fmt.Errorf("could not find PID in the entry in sheet Features, row:%s", row.properties["_row_in_sheet"])
		}

		if pid == "REST" {
			// Reached end of list, mark for each feature
			for prop, value := range row.properties {
				for f := range db.families {
					family := &db.families[f]
					for p := range family.Pids {
						pid_obj := &family.Pids[p]
						_, exists := pid_obj.Properties[prop]
						if !exists {
							pid_obj.Properties[prop] = value
							log.Printf("Setting feature default for %s, %s=%s\n",
								pid_obj.Properties["Internal name"], prop, value)
						}
					}
				}
			}
			continue
		}

		pid_obj := db.Get_pid_from_pid_name(pid)
		if pid_obj == nil {
			return fmt.Errorf("could not find PID %s in General sheet (referenced in row %s of Features sheet",
				pid, row.properties["_row_in_sheet"])
		}

		for prop, value := range row.properties {
			if len(value) > 0 {
				pid_obj.Properties[prop] = value
				log.Printf("Setting for %s feature %s=%s\n", pid_obj.Properties["Internal name"],
					prop, value)
			}
		}
	}

	f.Close()
	return nil
}
