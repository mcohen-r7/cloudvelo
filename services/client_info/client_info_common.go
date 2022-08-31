package client_info

import (
	"context"
	"time"

	cvelo_services "www.velocidex.com/golang/cloudvelo/services"
	crypto_proto "www.velocidex.com/golang/velociraptor/crypto/proto"
	"www.velocidex.com/golang/velociraptor/json"
)

var (
	// Query to retrieve all the task queued for a client.
	getClientTasksQuery = `{
  "sort": [
  {
    "timestamp": {"order": "asc"}
  }],
  "query": {
    "bool": {
      "must": [
         {"match": {"client_id" : %q}}
      ]}
  }
}
`
)

func (self ClientInfoManager) QueueMessageForClient(
	ctx context.Context, client_id string,
	req *crypto_proto.VeloMessage,
	notify bool, completion func()) error {

	return cvelo_services.SetElasticIndex(self.config_obj.OrgId,
		"tasks", "", &ClientTask{
			ClientId:  client_id,
			Timestamp: time.Now().UnixNano(),
			JSONData:  json.MustMarshalString(req),
		})
}

type ClientTask struct {
	ClientId  string `json:"client_id"`
	Timestamp int64  `json:"timestamp"`
	JSONData  string `json:"data"`
}

// Get the client's tasks and remove them from the queue.
func (self ClientInfoManager) GetClientTasks(
	ctx context.Context, client_id string) ([]*crypto_proto.VeloMessage, error) {

	query := json.Format(getClientTasksQuery, client_id)
	hits, err := cvelo_services.QueryElastic(ctx, self.config_obj.OrgId,
		"tasks", query)
	if err != nil {
		return nil, err
	}

	results := []*crypto_proto.VeloMessage{}
	for _, hit := range hits {
		err = cvelo_services.DeleteDocument(self.config_obj.OrgId,
			"tasks", hit.Id, cvelo_services.NoSync)
		if err != nil {
			return nil, err
		}

		item := &ClientTask{}
		err = json.Unmarshal(hit.JSON, item)
		if err != nil {
			continue
		}

		message := &crypto_proto.VeloMessage{}
		err = json.Unmarshal([]byte(item.JSONData), message)
		if err != nil {
			continue
		}
		results = append(results, message)
	}
	return results, nil
}