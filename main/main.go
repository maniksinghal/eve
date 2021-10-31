package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	webexteams "github.com/jbogarin/go-cisco-webex-teams/sdk"
	"github.com/magiconair/properties"
	query_parser "github.com/maniksinghal/eve/query_parser"
	stats "github.com/maniksinghal/eve/stats"
	schema_db "github.com/maniksinghal/eve/timing-db"
)

func respond_to_query(message string, roomId string, sender string) {

	var response string = query_parser.Parse_query(message, sender, my_db)

	markDownMessage := &webexteams.MessageCreateRequest{
		Text:   response,
		RoomID: roomId,
		//ToPersonID: person_id,
	}

	newMarkDownMessage, _, err := Client.Messages.CreateMessage(markDownMessage)
	if err != nil {
		fmt.Println("Error sending message " + err.Error())
	}
	fmt.Println("POST:", newMarkDownMessage.ID, newMarkDownMessage.Markdown,
		newMarkDownMessage.Created)
}

// create a handler struct
type HttpHandler struct{}

// implement `ServeHTTP` method on `HttpHandler` struct
func (h HttpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	fmt.Println("Bot received http request")

	/*
		for name, value := range req.Header {
			fmt.Printf("Header %s = %s\n", name, value)
		}
		fmt.Printf("Body: %s\n", req.Body)
	*/

	var result map[string]interface{}
	bodyBytes, _ := io.ReadAll(req.Body)

	json.Unmarshal(bodyBytes, &result)
	/*
		for name, value := range result {
			fmt.Printf("Body %s = %s\n", name, value)
		}
	*/

	_, exists := result["data"]
	if !exists {
		fmt.Println("Did not receive a valid message")
		return
	}

	var data map[string]interface{} = result["data"].(map[string]interface{})
	_, exists = data["id"]
	if !exists {
		fmt.Println("Did not receive valid data in message")
		return
	}

	_, exists = data["roomId"]
	if !exists {
		fmt.Println("Did not receive source of the message")
		return
	}

	_, exists = data["personEmail"]
	if !exists {
		fmt.Println("Unable to identify the query sender")
		return
	}

	// Get the message text
	message_id := data["id"].(string)
	room_id := data["roomId"].(string)
	sender := data["personEmail"].(string)

	if sender == bot_email {
		// Its me only
		fmt.Println("Ignored processing of my own reply")
		return
	}

	//fmt.Println("Extracted message-id: " + message_id)
	htmlMessageGet, _, err := Client.Messages.GetMessage(message_id)
	if err != nil {
		fmt.Println("Failed to extract message " + err.Error())
	}

	fmt.Println("Received query " + htmlMessageGet.Text + " from " +
		data["personEmail"].(string))

	respond_to_query(htmlMessageGet.Text, room_id, sender)

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
		fmt.Println("Error listing the webhooks")
	}
	for id, webhook := range webhooks.Items {
		fmt.Println("GET:", id, webhook.ID, webhook.Name, webhook.TargetURL, webhook.Created)

		// DELETE webhooks/<ID>
		_, err := Client.Webhooks.DeleteWebhook(webhook.ID)
		if err != nil {
			fmt.Println("Error deleting webhook " + err.Error())
		} else {
			fmt.Println("Deleted existing webhook " + webhook.Name)
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
			fmt.Println("Error creating webhook " + err.Error())
		}

		fmt.Println("POST:", testWebhook.ID, testWebhook.Name, testWebhook.TargetURL, testWebhook.Created)
	*/
}

var my_db schema_db.Schema_db

//var props *properties.Properties

func main() {

	props := properties.MustLoadFile("properties.config", properties.UTF8)

	fmt.Printf("Got %d properties\n", props.Len())
	for _, key := range props.Keys() {
		fmt.Printf("%s => %s\n", key, props.GetString(key, "null"))
	}

	// Initialize stats
	stats_db_user := props.GetString("stats_db_user", "anonymous")
	stats_db_pwd := props.GetString("stats_db_pwd", "blank")
	stats_db_host := props.GetString("stats_db_host", "localhost")
	stats_db_port := props.GetInt("stats_db_port", 0)
	var stats_handle stats.Stats_handle = new(stats.MySql_handle)
	stats_handle.Initialize(stats_db_user, stats_db_pwd, stats_db_host, stats_db_port)

	query_parser.Initialize()

	/*
		shellPtr := flag.Bool("shell", false, "Run in shell mode")
		flag.Parse()
	*/

	shell_mode := props.MustGetBool("shell_mode")
	fmt.Println("Got shell_mode: " + strconv.FormatBool(shell_mode))

	my_db = new(schema_db.Excel_db)
	schema_db.Parse_database(my_db, "Timing PIDs.xlsx")
	//schema_db.Dump_db(my_db)

	if !shell_mode {

		bot_token := props.MustGetString("bot_token")
		bot_email = props.MustGetString("bot_email")
		cleanup_webhooks(bot_token)

		// create a new handler
		handler := HttpHandler{}

		// listen and serve
		bot_listen_port := props.MustGetInt("bot_listen_port")
		bot_listen_path := fmt.Sprintf(":%d", bot_listen_port)
		fmt.Printf("Listening on port %d\n", bot_listen_port)
		http.ListenAndServe(bot_listen_path, handler)
		fmt.Println("Abrupt came out of listening!!")
	} else {
		/* Enter shell mode */
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Printf("query> ")
			text, _ := reader.ReadString('\n')
			text = strings.TrimSuffix(text, "\n")
			if text == "quit" {
				fmt.Println("Exiting the shell")
				break
			}
			output := query_parser.Parse_query(text, "shell_user", my_db)
			fmt.Println(output)
		}

	}

}
