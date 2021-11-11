package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	webexteams "github.com/jbogarin/go-cisco-webex-teams/sdk"
	"github.com/magiconair/properties"
	query_parser "github.com/maniksinghal/eve/query_parser"
	stats "github.com/maniksinghal/eve/stats"
	schema_db "github.com/maniksinghal/eve/timing-db"
	lumberjack "github.com/natefinch/lumberjack"
)

func respond_to_query(message string, roomId string, sender string) {

	var response string = query_parser.Parse_query(message, sender, &my_db)

	markDownMessage := &webexteams.MessageCreateRequest{
		Markdown: response,
		RoomID:   roomId,
		//ToPersonID: person_id,
	}

	newMarkDownMessage, _, err := Client.Messages.CreateMessage(markDownMessage)
	if err != nil {
		log.Println("Error sending message " + err.Error())
	}
	log.Println("POST:", newMarkDownMessage.ID, newMarkDownMessage.Markdown,
		newMarkDownMessage.Created)
}

// create a handler struct
type HttpHandler struct{}

// implement `ServeHTTP` method on `HttpHandler` struct
func (h HttpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	log.Println("Bot received http request")

	/*
		for name, value := range req.Header {
			log.Printf("Header %s = %s\n", name, value)
		}
		log.Printf("Body: %s\n", req.Body)
	*/

	var result map[string]interface{}
	bodyBytes, _ := io.ReadAll(req.Body)
	log.Println(bodyBytes)

	json.Unmarshal(bodyBytes, &result)
	/*
		for name, value := range result {
			log.Printf("Body %s = %s\n", name, value)
		}
	*/

	event_type, exists := result["resource"]

	_, exists = result["data"]
	if !exists {
		log.Println("Did not receive a valid message")
		return
	}

	var data map[string]interface{} = result["data"].(map[string]interface{})
	_, exists = data["id"]
	if !exists {
		log.Println("Did not receive valid data in message")
		return
	}

	_, exists = data["roomId"]
	if !exists {
		log.Println("Did not receive source of the message")
		return
	}

	_, exists = data["personEmail"]
	if !exists {
		log.Println("Unable to identify the query sender")
		return
	}

	// Get the message text
	message_id := data["id"].(string)
	room_id := data["roomId"].(string)
	sender := data["personEmail"].(string)

	var query string
	if sender == bot_email {

		if event_type == "memberships" {
			// Bot added to a room
			query = query_parser.Query_added_to_room
		} else {
			// Its me only
			log.Println("Ignored processing of my own reply")
			return
		}
	} else {

		//log.Println("Extracted message-id: " + message_id)
		htmlMessageGet, _, err := Client.Messages.GetMessage(message_id)
		if err != nil {
			log.Println("Failed to extract message " + err.Error())
		}

		query = htmlMessageGet.Text
		log.Println("Received query " + htmlMessageGet.Text + " from " +
			data["personEmail"].(string))
	}

	respond_to_query(query, room_id, sender)

	// create response binary data
	resp_data := []byte("Hello World!") // slice of bytes
	// write `data` to response
	res.Write(resp_data)
}

var Client *webexteams.Client
var bot_email string

/*
 * List and delete all existing webhooks from
 * the bot
 * Webhook creation is done by hookbuster
 */
func cleanup_webhooks(bot_id string) {
	Client = webexteams.NewClient()
	//bot_id = "MWU5ZGRmZmYtNDk4ZC00NTg1LWE0YmUtNTE2YjMyOWVhZGRjOWIxNmI2NmYtMGZj_PF84_1eb65fdf-9643-417f-9974-ad72cae0e10f"
	Client.SetAuthToken(bot_id)

	webhooksQueryParams := &webexteams.ListWebhooksQueryParams{
		Max: 10,
	}

	webhooks, _, err := Client.Webhooks.ListWebhooks(webhooksQueryParams)
	if err != nil {
		log.Println("Error listing the webhooks")
	}
	for id, webhook := range webhooks.Items {
		log.Println("GET:", id, webhook.ID, webhook.Name, webhook.TargetURL, webhook.Created)

		// DELETE webhooks/<ID>
		_, err := Client.Webhooks.DeleteWebhook(webhook.ID)
		if err != nil {
			log.Println("Error deleting webhook " + err.Error())
		} else {
			log.Println("Deleted existing webhook " + webhook.Name)
		}
	}

	// Webhooks are now created by hookbuster
	/*
		//myRoomID := ""                  // Change to your testing room
		//webHookURL := "https://abc.com" // Change this to your test URL
		webHookURL := "https://00c7-49-207-221-78.ngrok.io"

		// POST webhooks

		webhookRequest := &webexteams.WebhookCreateRequest{
			Name:      "Webhook - Test",
			TargetURL: webHookURL,
			Resource:  "messages",
			Event:     "created",
			//Filter:    "roomId=" + RoomID,
		}

		testWebhook, _, err := Client.Webhooks.CreateWebhook(webhookRequest)
		if err != nil {
			log.Println("Error creating webhook " + err.Error())
		}

		log.Println("POST:", testWebhook.ID, testWebhook.Name, testWebhook.TargetURL, testWebhook.Created)
	*/
}

var my_db schema_db.Schema_db

//var props *properties.Properties

func get_env_vars(props *properties.Properties) map[string]string {
	var env_map map[string]string = make(map[string]string)
	var env_var_map map[string]*string = make(map[string]*string)
	prop_keys := props.Keys()

	for _, key := range prop_keys {
		env_map[key] = props.MustGetString(key)
		env_var_map[key] = flag.String(key, "", "Refer properties.config")
	}
	flag.Parse()

	for _, key := range prop_keys {
		if *env_var_map[key] != "" {
			env_map[key] = *env_var_map[key]
		}
	}

	return env_map
}

func test_queries() {
	files, err := ioutil.ReadDir("test")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if !strings.Contains(f.Name(), "test_query") {
			continue
		}

		file_name := fmt.Sprintf("test/%s", f.Name())
		fmt.Printf("Testing query: %s\n", file_name)

		bytes, err := os.ReadFile(file_name)
		if err != nil {
			panic(err)
		}

		lines := strings.Split(string(bytes[:]), "\n")
		query := lines[0]
		expected_output := strings.Join(lines[1:], "\n")
		output := query_parser.Parse_query(query, "test_user", &my_db)
		if output != expected_output {
			fmt.Printf("%s: %s => Failed\n", file_name, query)
			fmt.Printf("Expected: \n'%s'\n\n\nFound: \n'%s'\n", expected_output, output)
			return
		} else {
			fmt.Printf("%s: %s => Passed\n", file_name, query)
		}

	}
}

func create_test(query string, output string) {
	count := 0
	max := 10000
	var file_name string
	for count < max {
		file_name = fmt.Sprintf("test/test_query_%04d", count)
		if _, err := os.Stat(file_name); errors.Is(err, os.ErrNotExist) {
			break
		} else {
			count += 1
		}
	}

	if count == max {
		log.Printf("Could not generate test-case file name")
		return
	}

	f, err := os.Create(file_name)
	if err != nil {
		log.Printf("Could not create test case file - %s\n", err.Error())
		return
	}

	f.WriteString(query)
	f.WriteString("\n")
	f.WriteString(output)
	f.Close()
	fmt.Printf("Created test case %s with query %s", file_name, query)
}

func main() {

	log.SetOutput(&lumberjack.Logger{
		Filename:   "./eve.log",
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     28,
	})

	props := properties.MustLoadFile("properties.config", properties.UTF8)
	env_map := get_env_vars(props)

	log.Printf("Got %d properties\n", props.Len())
	for _, key := range props.Keys() {
		log.Printf("%s => %s/%s\n", key, env_map[key],
			props.GetString(key, "null"))
	}

	// Initialize stats
	stats_db_user := env_map["stats_db_user"]
	stats_db_pwd := env_map["stats_db_pwd"]
	stats_db_host := env_map["stats_db_host"]
	stats_db_port, err := strconv.Atoi(env_map["stats_db_port"])
	if err != nil {
		panic(err)
	}
	stats_mode := env_map["stats_mode"]
	timing_db_file := env_map["timing_db_file"]
	bot_token := env_map["bot_token"]
	bot_email = env_map["bot_email"]
	bot_listen_port, err := strconv.Atoi(env_map["bot_listen_port"])
	if err != nil {
		panic(err)
	}

	var stats_handle stats.Stats_handle
	if stats_mode == "file" {
		stats_handle = new(stats.File_stats)
	} else if stats_mode == "mysql" {
		stats_handle = new(stats.MySql_handle)
	}
	stats_handle.Initialize(stats_db_user, stats_db_pwd, stats_db_host, stats_db_port)

	shell_mode, err := strconv.ParseBool(env_map["shell_mode"])
	if err != nil {
		panic(err)
	}
	log.Println("Got shell_mode: " + strconv.FormatBool(shell_mode))

	my_db = new(schema_db.Excel_db)
	schema_db.Parse_database(my_db, timing_db_file)

	query_parser.Initialize()

	/*
	 * Check if test mode
	 */
	test_mode, err := strconv.ParseBool(env_map["test"])
	if err != nil {
		panic(err)
	}

	if test_mode {
		test_queries()
		return
	}

	if !shell_mode {

		cleanup_webhooks(bot_token)

		// create a new handler
		handler := HttpHandler{}

		// listen and serve

		bot_listen_path := fmt.Sprintf(":%d", bot_listen_port)
		log.Printf("Listening on port %d\n", bot_listen_port)
		http.ListenAndServe(bot_listen_path, handler)
		log.Println("Abrupt came out of listening!!")
	} else {
		/* Enter shell mode */
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Printf("query> ")

			text, _ := reader.ReadString('\n')
			text = strings.TrimSuffix(text, "\n")
			if strings.EqualFold(text, "quit") || strings.EqualFold(text, "exit") {
				log.Println("Exiting the shell")
				break
			}
			output := query_parser.Parse_query(text, "shell_user", &my_db)
			if strings.Contains(text, "create_test:") {
				text = strings.Split(text, ":")[1]
				create_test(text, output)
			}
			fmt.Println(output)
		}

	}

}
