package aliyun

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

// ---------------------------------------------------------------------------
// Kafka (alikafka) instances
//   GetInstanceList → GetTopicList / DescribeSaslUsers / GetConsumerList
//   DescribeAcls is queried per SASL user (needs Username + resource filters)
// ---------------------------------------------------------------------------

type KafkaFetcher struct{}

func (f *KafkaFetcher) ResourceType() string { return "kfk" }

func (f *KafkaFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchKafkaInstances(ctx, p)
}

type kafkaInstanceListResponse struct {
	InstanceList struct {
		InstanceVO []kafkaInstance `json:"InstanceVO"`
	} `json:"InstanceList"`
}

type kafkaInstance struct {
	InstanceID      string `json:"InstanceId"`
	Name            string `json:"Name"`
	RegionID        string `json:"RegionId"`
	ServiceStatus   int    `json:"ServiceStatus"`
	SpecType        string `json:"SpecType"`
	Series          string `json:"Series"`
	PaidType        int    `json:"PaidType"`
	VpcID           string `json:"VpcId"`
	CreateTime      int64  `json:"CreateTime"`
	ExpiredTime     int64  `json:"ExpiredTime"`
	UsedTopicCount  int    `json:"UsedTopicCount"`
	UsedGroupCount  int    `json:"UsedGroupCount"`
	DomainEndpoint  string `json:"DomainEndpoint"`
	SaslEndPoint    string `json:"SaslEndPoint"`
	AllConfig       string `json:"AllConfig"`
	MsgRetain       int    `json:"MsgRetain"`
	StandardZoneID  string `json:"StandardZoneId"`
	SecurityGroup   string `json:"SecurityGroup"`
	ResourceGroupID string `json:"ResourceGroupId"`
}

type kafkaEnriched struct {
	kafkaInstance
	Topics        []kafkaTopic      `json:"topics"`
	SASLUsers     []kafkaSASLUser   `json:"sasl_users"`
	ConsumerGroup []kafkaConsumer   `json:"consumer_groups"`
	ACLs          []kafkaACL        `json:"acls"`
	AllConfigMap  map[string]string `json:"all_config"`
}

type kafkaTopic struct {
	Topic        string `json:"Topic"`
	PartitionNum int    `json:"PartitionNum"`
	Status       int    `json:"Status"`
	AutoCreate   bool   `json:"AutoCreate"`
	CompactTopic bool   `json:"CompactTopic"`
	CreateTime   int64  `json:"CreateTime"`
	Remark       string `json:"Remark"`
}

type kafkaSASLUser struct {
	Username  string `json:"Username"`
	Type      string `json:"Type"`
	Mechanism string `json:"Mechanism"`
}

type kafkaConsumer struct {
	ConsumerGroup string            `json:"ConsumerGroup"`
	Remark        string            `json:"Remark"`
	Topics        map[string]string `json:"topics"` // topic → total accumulations
}

type kafkaACL struct {
	Username              string `json:"Username"`
	AclResourceType       string `json:"AclResourceType"`
	AclResourceName       string `json:"AclResourceName"`
	AclOperationType      string `json:"AclOperationType"`
	AclPermissionType     string `json:"AclPermissionType"`
	AclResourcePatternType string `json:"AclResourcePatternType"`
	Host                  string `json:"Host"`
}

var kafkaServiceStatus = map[int]string{
	0:   "Deploying",
	2:   "Starting",
	3:   "Running",
	5:   "Running",
	101: "Upgrading",
	102: "Changing",
	-1:  "DeployFailed",
	-2:  "Expired",
	-3:  "Released",
}

func fetchKafkaInstances(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	regions := p.ProfileRegions[p.ActiveProfile]
	if len(regions) == 0 {
		regions = []string{p.Regions[0]}
	}

	for _, region := range regions {
		args := []string{"alikafka", "GetInstanceList", "--RegionId", region}
		out, err := runAliyun(ctx, args, p.ActiveProfile)
		if err != nil {
			continue
		}

		var resp kafkaInstanceListResponse
		if json.Unmarshal(out, &resp) != nil {
			continue
		}

		for _, inst := range resp.InstanceList.InstanceVO {
			enriched := enrichKafkaInstance(ctx, inst, region, p.ActiveProfile)

			rawJSON, _ := json.Marshal(enriched)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "kfk",
				Region:       region,
				Profile:      p.ActiveProfile,
				ResourceID:   inst.InstanceID,
				ResourceName: inst.Name,
				Status:       kafkaServiceStatus[inst.ServiceStatus],
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}
	}

	return allResources, nil
}

func enrichKafkaInstance(ctx context.Context, inst kafkaInstance, region, profile string) kafkaEnriched {
	e := kafkaEnriched{kafkaInstance: inst}

	// Parse AllConfig JSON string into a map
	if inst.AllConfig != "" {
		var m map[string]string
		if json.Unmarshal([]byte(inst.AllConfig), &m) == nil {
			e.AllConfigMap = m
		}
	}

	// Topics
	{
		args := []string{"alikafka", "GetTopicList", "--RegionId", region, "--InstanceId", inst.InstanceID}
		if out, err := runAliyun(ctx, args, profile); err == nil {
			var resp struct {
				TopicList struct {
					TopicVO []kafkaTopic `json:"TopicVO"`
				} `json:"TopicList"`
			}
			if json.Unmarshal(out, &resp) == nil {
				e.Topics = resp.TopicList.TopicVO
			}
		}
	}

	// SASL users — NOTE: the API returns plaintext passwords; we keep only
	// the username and never persist the Password field.
	{
		args := []string{"alikafka", "DescribeSaslUsers", "--RegionId", region, "--InstanceId", inst.InstanceID}
		if out, err := runAliyun(ctx, args, profile); err == nil {
			var resp struct {
				SaslUserList struct {
					SaslUserVO []struct {
						Username  string `json:"Username"`
						Type      string `json:"Type"`
						Mechanism string `json:"Mechanism"`
					} `json:"SaslUserVO"`
				} `json:"SaslUserList"`
			}
			if json.Unmarshal(out, &resp) == nil {
				for _, u := range resp.SaslUserList.SaslUserVO {
					e.SASLUsers = append(e.SASLUsers, kafkaSASLUser{
						Username: u.Username, Type: u.Type, Mechanism: u.Mechanism,
					})
				}
			}
		}
	}

	// Consumer groups
	{
		args := []string{"alikafka", "GetConsumerList", "--RegionId", region, "--InstanceId", inst.InstanceID}
		if out, err := runAliyun(ctx, args, profile); err == nil {
			var resp struct {
				ConsumerList struct {
					ConsumerVO []struct {
						ConsumerGroup string `json:"ConsumerGroup"`
						Remark        string `json:"Remark"`
					} `json:"ConsumerVO"`
				} `json:"ConsumerList"`
			}
			if json.Unmarshal(out, &resp) == nil {
				for _, c := range resp.ConsumerList.ConsumerVO {
					e.ConsumerGroup = append(e.ConsumerGroup, kafkaConsumer{
						ConsumerGroup: c.ConsumerGroup, Remark: c.Remark,
					})
				}
			}
		}
	}

	// ACLs — DescribeAcls requires Username + AclResourceType + AclResourceName
	// per call; iterate non-default SASL users with Topic resource and "*" name.
	for _, u := range e.SASLUsers {
		if u.Username == inst.InstanceID {
			continue // instance default account, not a custom SASL user
		}
		args := []string{"alikafka", "DescribeAcls",
			"--RegionId", region,
			"--InstanceId", inst.InstanceID,
			"--Username", u.Username,
			"--AclResourceType", "Topic",
			"--AclResourceName", "*",
		}
		out, err := runAliyun(ctx, args, profile)
		if err != nil {
			continue
		}
		var resp struct {
			KafkaAclList struct {
				KafkaAclVO []kafkaACL `json:"KafkaAclVO"`
			} `json:"KafkaAclList"`
		}
		if json.Unmarshal(out, &resp) == nil {
			e.ACLs = append(e.ACLs, resp.KafkaAclList.KafkaAclVO...)
		}
	}

	// Stable ordering for display
	sort.Slice(e.Topics, func(i, j int) bool { return e.Topics[i].Topic < e.Topics[j].Topic })
	sort.Slice(e.ACLs, func(i, j int) bool {
		if e.ACLs[i].Username != e.ACLs[j].Username {
			return e.ACLs[i].Username < e.ACLs[j].Username
		}
		return e.ACLs[i].AclOperationType < e.ACLs[j].AclOperationType
	})

	return e
}
