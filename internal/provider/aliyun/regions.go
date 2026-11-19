package aliyun

import "github.com/mars-base/cloudres/internal/core"

// regionOptions is the static Aliyun region catalog used by the CLI warning
// and the TUI region picker, so neither path needs a live DescribeRegions
// call. Transcribed from `aliyun ecs DescribeRegions` output; refresh it when
// Aliyun publishes new regions.
var regionOptions = []core.RegionOption{
	{ID: "cn-qingdao", Name: "华北1（青岛）"},
	{ID: "cn-beijing", Name: "华北2（北京）"},
	{ID: "cn-zhangjiakou", Name: "华北3（张家口）"},
	{ID: "cn-huhehaote", Name: "华北5（呼和浩特）"},
	{ID: "cn-wulanchabu", Name: "华北6（乌兰察布）"},
	{ID: "cn-hangzhou", Name: "华东1（杭州）"},
	{ID: "cn-shanghai", Name: "华东2（上海）"},
	{ID: "cn-nanjing", Name: "华东5（南京-本地地域）"},
	{ID: "cn-fuzhou", Name: "华东6（福州-本地地域）"},
	{ID: "cn-shenzhen", Name: "华南1（深圳）"},
	{ID: "cn-heyuan", Name: "华南2（河源）"},
	{ID: "cn-guangzhou", Name: "华南3（广州）"},
	{ID: "cn-wuhan-lr", Name: "华中1（武汉-本地地域）"},
	{ID: "cn-chengdu", Name: "西南1（成都）"},
	{ID: "cn-zhongwei", Name: "西北2（中卫）"},
	{ID: "cn-hongkong", Name: "中国香港"},
	{ID: "ap-northeast-1", Name: "日本（东京）"},
	{ID: "ap-northeast-2", Name: "韩国（首尔）"},
	{ID: "ap-southeast-1", Name: "新加坡"},
	{ID: "ap-southeast-3", Name: "马来西亚（吉隆坡）"},
	{ID: "ap-southeast-5", Name: "印度尼西亚（雅加达）"},
	{ID: "ap-southeast-6", Name: "菲律宾（马尼拉）"},
	{ID: "ap-southeast-7", Name: "泰国（曼谷）"},
	{ID: "ap-southeast-8", Name: "马来西亚（柔佛州）"},
	{ID: "us-east-1", Name: "美国（弗吉尼亚）"},
	{ID: "us-west-1", Name: "美国（硅谷）"},
	{ID: "na-south-1", Name: "墨西哥"},
	{ID: "sa-east-1", Name: "巴西（圣保罗）"},
	{ID: "eu-west-1", Name: "英国（伦敦）"},
	{ID: "eu-west-2", Name: "法国（巴黎）"},
	{ID: "eu-west-3", Name: "荷兰（阿姆斯特丹）"},
	{ID: "eu-central-1", Name: "德国（法兰克福）"},
	{ID: "me-east-1", Name: "阿联酋（迪拜）"},
}

// regionScopedTypes lists Aliyun resource types that can only be queried with
// an explicit region. Types absent from this set (oss, ram, acr, cms-*,
// arms-* contacts) are account-global and work without one.
var regionScopedTypes = map[string]bool{
	"ecs": true, "vpc": true, "vsw": true, "rds": true, "tair": true,
	"pdb": true, "slb": true, "alb": true, "nlb": true, "ess": true,
	"cas": true, "ack": true, "acl": true, "kfk": true, "ck": true,
	"prom": true,
}

// defaultRegion is used as the CLI endpoint region for account-global APIs
// (ram, acr, cms-*, arms-*) when the active profile has no region configured;
// the aliyun CLI itself rejects calls with "region can't be empty".
const defaultRegion = "cn-hangzhou"

// regionOverride is appended to every aliyun CLI call as the global --region
// flag. Set from the CLI region gate or the TUI picker before querying.
var regionOverride string

func (d *Detector) RegionOptions() []core.RegionOption { return regionOptions }

func (d *Detector) RequiresRegion(resourceType string) bool {
	return regionScopedTypes[resourceType]
}

func (d *Detector) SetRegion(region string) {
	if region == "" {
		region = defaultRegion
	}
	regionOverride = region
}

var _ core.RegionSupport = (*Detector)(nil)
