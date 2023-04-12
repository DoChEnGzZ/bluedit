/**
    @author:huchao
    @data:2022/2/15
    @note:
**/
package redis

import (
	"bluebell_backend/models"
	"errors"
	"github.com/go-redis/redis"
	"go.uber.org/zap"
	"strconv"
	"time"
)

/**
 * @Author huchao
 * @Description //TODO redis存储帖子信息
 * @Date 17:08 2022/2/14
 **/
// CreatePost 使用hash存储帖子信息
func CreatePost(postID, userID uint64, title, summary string, CommunityID uint64) (err error) {
	now := float64(time.Now().Unix())
	votedKey := KeyPostVotedZSetPrefix + strconv.Itoa(int(postID))
	communityKey := KeyCommunityPostSetPrefix + strconv.Itoa(int(CommunityID))
	postInfo := map[string]interface{}{
		"title":    title,
		"summary":  summary,
		"post:id":  postID,
		"user:id":  userID,
		"time":     now,
		"votes":    1,
		"comments": 0,
		"community:id": CommunityID,
	}

	// 事务操作
	pipeline := client.TxPipeline()
	pipeline.ZAdd(votedKey, redis.Z{ // 作者默认投赞成票
		Score:  1,
		Member: userID,
	})
	pipeline.Expire(votedKey, time.Second*OneWeekInSeconds) // 一周时间

	pipeline.HMSet(KeyPostInfoHashPrefix+strconv.Itoa(int(postID)), postInfo)
	//缓存过期时间：2个小时
	pipeline.Expire(KeyPostInfoHashPrefix+strconv.Itoa(int(postID)),time.Hour*2)
	pipeline.ZAdd(KeyPostScoreZSet, redis.Z{ // 添加到分数的ZSet
		Score:  now + VoteScore,
		Member: postID,
	})
	pipeline.ZAdd(KeyPostTimeZSet, redis.Z{ // 添加到时间的ZSet
		Score:  now,
		Member: postID,
	})
	pipeline.SAdd(communityKey, postID) // 添加到对应版块  把帖子添加到社区的set
	_, err = pipeline.Exec()
	return
}

// GetPost 从key中分页取出帖子
func GetPost(order string, page int64) []map[string]string {
	key := KeyPostScoreZSet
	if order == "time" {
		key = KeyPostTimeZSet
	}
	start := (page - 1) * PostPerAge
	end := start + PostPerAge - 1
	ids := client.ZRevRange(key, start, end).Val()
	postList := make([]map[string]string, 0, len(ids))
	for _, id := range ids {
		postData := client.HGetAll(KeyPostInfoHashPrefix + id).Val()
		postData["id"] = id
		postList = append(postList, postData)
	}
	return postList
}

func CreatePostCache(postID, userID uint64, title, summary string, CommunityID uint64,createTime time.Time) (err error) {
	postInfo := map[string]interface{}{
		"title":    title,
		"summary":  summary,
		"post:id":  postID,
		"user:id":  userID,
		"time":     createTime,
		"votes":    1,
		"comments": 0,
	}
	pipeLine:=client.Pipeline()
	pipeLine.HMSet(KeyPostInfoHashPrefix+strconv.Itoa(int(postID)),postInfo)
	//缓存过期时间：2个小时
	pipeLine.Expire(KeyPostInfoHashPrefix+strconv.Itoa(int(postID)),time.Hour*2)
	_, err = pipeLine.Exec()
	if err != nil {
		return err
	}
	return nil
}

func DeletePostCache(postID uint64)error{
	pipeLine:=client.Pipeline()
	pipeLine.Del(KeyPostInfoHashPrefix+strconv.Itoa(int(postID)))
	_, err := pipeLine.Exec()
	if err != nil {
		return err
	}
	exist, _ := client.Exists(KeyPostInfoHashPrefix+strconv.Itoa(int(postID))).Result()
	zap.L().Debug("delete post cache",zap.Uint64("postID",postID),zap.Bool("exist",exist>1))
	if err != nil {
		return err
	}
	return nil
}

/*
 *GetPostByID
 *@Description: 根据帖子ID从redis缓存中获取帖子，帖子中无
 *@param id
 *@return postList
 *@return err
*/
func GetPostByID(id string) (post *models.Post, err error){
	postKey:=KeyPostInfoHashPrefix+id
	if client.Exists(postKey).Val()<1{
		return nil,errors.New("cache on "+id+" dont hit")
	}
	postData:=client.HGetAll(postKey).Val()
	post_id,_:=strconv.Atoi(id)
	author_id,_:=strconv.Atoi(postData["user:id"])
	votes,_:=strconv.ParseInt(postData["votes"],10,64)
	u,_:=strconv.ParseInt(postData["time"],10,64)
	community_id,_:=strconv.ParseInt(postData["community:id"],10,64)
	//postInfo := map[string]interface{}{
	//	"title":    title,
	//	"summary":  summary,
	//	"post:id":  postID,
	//	"user:id":  userID,
	//	"time":     now,
	//	"votes":    1,
	//	"comments": 0,
	//}
	t:=time.Unix(u,0)
	post=&models.Post{
		PostID:      uint64(post_id),
		AuthorId:    uint64(author_id),
		Title:       postData["title"],
		CommunityID: uint64(community_id),
		Summary:     postData["summary"],
		CreateTime:  t,
		VoteNum:     votes ,
	}
	return post,nil
}

func UpdatePostCache(postID, userID uint64, title, summary string, CommunityID uint64,createTime time.Time) (err error)  {
	//更新缓存前先删除缓存
	//err=DeletePostCache(postID)
	//if err!=nil{
	//	return err
	//}
	//zap.L().Debug("update post cache",zap.Any("communityID",strconv.Itoa(int(CommunityID))))
	postInfo:= map[string]interface{}{
		"title":    title,
		"summary":  summary,
		"post:id":  strconv.Itoa(int(postID)),
		"user:id":  strconv.Itoa(int(userID)),
		"community:id":strconv.Itoa(int(CommunityID)),
		"time":     strconv.FormatInt(createTime.Unix(),10),
	}
	//zap.L().Debug("update post cache",zap.Any("postInfo",postInfo))
	pipeline:=client.Pipeline()
	//更新redis缓存
	pipeline.HMSet(KeyPostInfoHashPrefix+strconv.Itoa(int(postID)),postInfo)
//缓存过期时间：2个小时
	pipeline.Expire(KeyPostInfoHashPrefix+strconv.Itoa(int(postID)),time.Hour*2)
	_, err = pipeline.Exec()
	if err != nil {
		return err
	}
	return nil
}

// GetCommunityPost 分社区根据发帖时间或者分数取出分页的帖子
func GetCommunityPost(communityName, orderKey string, page int64) []map[string]string {
	key := orderKey + communityName // 创建缓存键

	if client.Exists(key).Val() < 1 {
		client.ZInterStore(key, redis.ZStore{
			Aggregate: "MAX",
		}, KeyCommunityPostSetPrefix+communityName, orderKey)
		client.Expire(key, 60*time.Second)
	}
	return GetPost(key, page)
}

/**
 * @Author huchao
 * @Description //TODO 按照分数从大到小的顺序查询指定数量的元素
 * @Date 0:12 2022/2/17
 **/
func getIDsFormKey(key string, page, size int64) ([]string, error) {
	start := (page-1) * size
	end := start + size - 1
	//zap.L().Debug("getIDsFormKey",zap.String("key",key),zap.Int64("start",start),zap.Int64("end",end))
	// 3.ZREVRANGE 按照分数从大到小的顺序查询指定数量的元素

	return client.ZRevRange(key, start, end).Result()
}

/**
 * @Author huchao
 * @Description //TODO 升级版投票列表接口：按创建时间排序 或者 按照 分数排序 (查询出的ids已经根据order从大到小排序)
 * @Date 22:19 2022/2/15
 **/
func GetPostIDsInOrder(p *models.ParamPostList) ([] string, error)  {
	// 从redis获取id
	// 1.根据用户请求中携带的order参数确定要查询的redis key
	key := KeyPostTimeZSet		// 默认是时间
	if p.Order == models.OrderScore {	// 按照分数请求
		key = KeyPostScoreZSet
	}
	// 2.确定查询的索引起始点
	zap.L().Debug("page",zap.Int64("page",p.Page),zap.Int64("size",p.Size))
	ids,err:=getIDsFormKey(key, p.Page ,p.Size)
	if err!=nil{
		return nil, err
	}
	//zap.L().Debug("ids",zap.Any("ids",ids))
	return ids, nil
}

/**
 * @Author huchao
 * @Description //TODO 根据ids查询每篇帖子的投赞成票的数据
 * @Date 21:28 2022/2/16
 **/
func GetPostVoteData(ids []string) (data []int64, err error)  {
	//data = make([]int64, 0, len(ids))
	//for _, id := range ids{
	//	key := KeyPostVotedZSetPrefix + id
	//	// 查找key中分数是1的元素数量 -> 统计每篇帖子的赞成票的数量
	//	v := client.ZCount(key, "1", "1").Val()
	//	data = append(data, v)
	//}
	// 使用 pipeline一次发送多条命令减少RTT
	pipeline := client.Pipeline()
	for _, id := range ids{
		//key := KeyCommunityPostSetPrefix + id
		key:=KeyPostVotedZSetPrefix+id
		pipeline.ZCount(key, "1", "1")
		//zap.L().Debug("get post vote",zap.Any("id",id),
		//	zap.Any("votes",client.ZCount(key,"1","1").Val()))
	}
	cmders, err := pipeline.Exec()
	if err != nil {
		return nil, err
	}
	data = make([]int64, 0, len(cmders))
	for _, cmder := range cmders{
		v := cmder.(*redis.IntCmd).Val()
		data = append(data, v)
	}
	return
}

/**
 * @Author huchao
 * @Description //TODO 按社区查询ids(查询出的ids已经根据order从大到小排序)
 * @Date 23:06 2022/2/16
 * @Param orderKey:按照分数或时间排序
	将社区key与orderkey(社区或时间)做zinterstore
 **/
func GetCommunityPostIDsInOrder(p *models.ParamPostList) ([]string, error) {
	// 1.根据用户请求中携带的order参数确定要查询的redis key
	orderkey := KeyPostTimeZSet		// 默认是时间
	if p.Order == models.OrderScore {	// 按照分数请求
		orderkey = KeyPostScoreZSet
	}

	// 使用zinterstore 把分区的帖子set与帖子分数的zset生成一个新的zset
	// 针对新的zset 按之前的逻辑取数据

	// 社区的key
	cKey := KeyCommunityPostSetPrefix + strconv.Itoa(int(p.CommunityID))

	// 利用缓存key减少zinterstore执行的次数 缓存key
	key := orderkey + strconv.Itoa(int(p.CommunityID))
	if client.Exists(key).Val() < 1 {
		// 不存在，需要计算
		pipeline := client.Pipeline()
		pipeline.ZInterStore(key, redis.ZStore{
			Aggregate: "MAX",	// 将两个zset函数聚合的时候 求最大值
		}, cKey, orderkey)		// zinterstore 计算
		pipeline.Expire(key, 60*time.Second)	// 设置超时时间
		_, err := pipeline.Exec()
		if err != nil {
			return nil, err
		}
	}
	// 存在的就直接根据key查询ids
	return getIDsFormKey(key ,p.Page, p.Size)
}