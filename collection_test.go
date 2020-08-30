package qmgo

import (
	"context"
	"github.com/qiniu/qmgo/operator"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
)

func TestCollection_EnsureIndex(t *testing.T) {
	ast := require.New(t)
	cli := initClient("test")
	defer cli.Close(context.Background())
	defer cli.DropCollection(context.Background())

	cli.ensureIndex(context.Background(), nil, false)
	cli.ensureIndex(context.Background(), []string{"id1"}, true)
	cli.ensureIndex(context.Background(), []string{"id2,id3"}, false)
	cli.ensureIndex(context.Background(), []string{"id4,-id5"}, false)

	// same index，panic
	ast.Panics(func() { cli.ensureIndex(context.Background(), []string{"id1"}, false) })

	// check if unique indexs is working
	var err error
	doc := bson.M{
		"id1": 1,
	}
	_, err = cli.InsertOne(context.Background(), doc)
	ast.NoError(err)

	coll, err := cli.CloneCollection()
	ast.NoError(err)
	_, err = coll.InsertOne(context.Background(), doc)
	ast.Equal(true, IsDup(err))
}

func TestCollection_EnsureIndexes(t *testing.T) {
	ast := require.New(t)
	cli := initClient("test")
	defer cli.Close(context.Background())
	defer cli.DropCollection(context.Background())

	unique := []string{"id1"}
	common := []string{"id2,id3", "id4,-id5"}
	cli.EnsureIndexes(context.Background(), unique, common)

	// same index，panic
	ast.Panics(func() { cli.EnsureIndexes(context.Background(), nil, unique) })

	// check if unique indexs is working
	var err error
	doc := bson.M{
		"id1": 1,
	}

	_, err = cli.InsertOne(context.Background(), doc)
	ast.NoError(err)
	_, err = cli.InsertOne(context.Background(), doc)
	ast.Equal(true, IsDup(err))
}

func TestCollection_DropIndex(t *testing.T) {
	ast := require.New(t)

	cli := initClient("test")
	defer cli.DropCollection(context.Background())

	cli.ensureIndex(context.Background(), []string{"index1"}, true)

	// same index，panic
	ast.Panics(func() { cli.ensureIndex(context.Background(), []string{"index1"}, false) })

	err := cli.DropIndexes(context.Background(), []string{"index1"})
	ast.NoError(err)
	ast.NotPanics(func() { cli.ensureIndex(context.Background(), []string{"index1"}, false) })

	cli.ensureIndex(context.Background(), []string{"-index1"}, true)
	// same index，panic
	ast.Panics(func() { cli.ensureIndex(context.Background(), []string{"-index1"}, false) })

	err = cli.DropIndexes(context.Background(), []string{"-index1"})
	ast.NoError(err)
	ast.NotPanics(func() { cli.ensureIndex(context.Background(), []string{"-index1"}, false) })

	err = cli.DropIndexes(context.Background(), []string{""})
	ast.Error(err)

	err = cli.DropIndexes(context.Background(), []string{"index2"})
	ast.Error(err)

	cli.ensureIndex(context.Background(), []string{"index2,-index1"}, true)
	ast.Panics(func() { cli.ensureIndex(context.Background(), []string{"index2,-index1"}, false) })
	err = cli.DropIndexes(context.Background(), []string{"index2,-index1"})
	ast.NoError(err)
	ast.NotPanics(func() { cli.ensureIndex(context.Background(), []string{"index2,-index1"}, false) })

	err = cli.DropIndexes(context.Background(), []string{"-"})
	ast.Error(err)
}

func TestCollection_Insert(t *testing.T) {
	ast := require.New(t)

	cli := initClient("test")

	defer cli.Close(context.Background())
	defer cli.DropCollection(context.Background())

	cli.EnsureIndexes(context.Background(), []string{"name"}, nil)

	var err error
	doc := bson.M{"_id": primitive.NewObjectID(), "name": "Alice"}

	res, err := cli.InsertOne(context.Background(), doc)
	ast.NoError(err)
	ast.NotEmpty(res)
	ast.Equal(doc["_id"], res.InsertedID)

	res, err = cli.InsertOne(context.Background(), doc)
	ast.Equal(true, IsDup(err))
	ast.Empty(res)
}

func TestCollection_InsertMany(t *testing.T) {
	ast := require.New(t)
	cli := initClient("test")
	defer cli.Close(context.Background())
	defer cli.DropCollection(context.Background())
	cli.EnsureIndexes(context.Background(), []string{"name"}, nil)

	var err error
	newDocs := []UserInfo{{Name: "Alice", Age: 10}, {Name: "Lucas", Age: 11}}
	res, err := cli.InsertMany(context.Background(), newDocs)
	ast.NoError(err)
	ast.NotEmpty(res)
	ast.Equal(2, len(res.InsertedIDs))

	newPDocs := []*UserInfo{{Name: "Alice03", Age: 10}, {Name: "Lucas03", Age: 11}}
	res, err = cli.InsertMany(context.Background(), newPDocs)
	ast.NoError(err)
	ast.NotEmpty(res)
	ast.Equal(2, len(res.InsertedIDs))

	docs2 := []UserInfo{
		{Name: "Alice"},
		{Name: "Lucas"},
	}
	res, err = cli.InsertMany(context.Background(), docs2)
	ast.Equal(true, IsDup(err))
	ast.Equal(0, len(res.InsertedIDs))

	docs4 := []UserInfo{}
	res, err = cli.InsertMany(context.Background(), []interface{}{docs4})
	ast.Error(err)
	ast.Empty(res)

}

func TestCollection_Upsert(t *testing.T) {
	ast := require.New(t)
	cli := initClient("test")
	defer cli.Close(context.Background())
	defer cli.DropCollection(context.Background())
	cli.EnsureIndexes(context.Background(), []string{"name"}, nil)

	id1 := primitive.NewObjectID()
	id2 := primitive.NewObjectID()
	docs := []interface{}{
		bson.D{{Key: "_id", Value: id1}, {Key: "name", Value: "Alice"}},
		bson.D{{Key: "_id", Value: id2}, {Key: "name", Value: "Lucas"}},
	}
	_, _ = cli.InsertMany(context.Background(), docs)

	var err error

	// replace already exist
	filter1 := bson.M{
		"name": "Alice",
	}
	replacement1 := bson.M{
		"name": "Alice1",
		"age":  18,
	}
	res, err := cli.Upsert(context.Background(), filter1, replacement1)
	ast.NoError(err)
	ast.NotEmpty(res)
	ast.Equal(int64(1), res.MatchedCount)
	ast.Equal(int64(1), res.ModifiedCount)
	ast.Equal(int64(0), res.UpsertedCount)
	ast.Equal(nil, res.UpsertedID)

	// not exist
	filter2 := bson.M{
		"name": "Lily",
	}
	replacement2 := bson.M{
		"name": "Lily",
		"age":  20,
	}
	res, err = cli.Upsert(context.Background(), filter2, replacement2)
	ast.NoError(err)
	ast.NotEmpty(res)
	ast.Equal(int64(0), res.MatchedCount)
	ast.Equal(int64(0), res.ModifiedCount)
	ast.Equal(int64(1), res.UpsertedCount)
	ast.NotNil(res.UpsertedID)

	// filter is nil or wrong BSON Document format
	replacement3 := bson.M{
		"name": "Geek",
		"age":  21,
	}
	res, err = cli.Upsert(context.Background(), nil, replacement3)
	ast.Error(err)
	ast.Empty(res)

	res, err = cli.Upsert(context.Background(), 1, replacement3)
	ast.Error(err)
	ast.Empty(res)

	// replacement is nil or wrong BSON Document format
	filter4 := bson.M{
		"name": "Geek",
	}
	res, err = cli.Upsert(context.Background(), filter4, nil)
	ast.Error(err)
	ast.Empty(res)

	res, err = cli.Upsert(context.Background(), filter4, 1)
	ast.Error(err)
	ast.Empty(res)
}

func TestCollection_Update(t *testing.T) {
	ast := require.New(t)
	cli := initClient("test")
	defer cli.Close(context.Background())
	defer cli.DropCollection(context.Background())
	cli.EnsureIndexes(context.Background(), []string{"name"}, nil)

	id1 := primitive.NewObjectID()
	id2 := primitive.NewObjectID()
	docs := []interface{}{
		bson.D{{Key: "_id", Value: id1}, {Key: "name", Value: "Alice"}},
		bson.D{{Key: "_id", Value: id2}, {Key: "name", Value: "Lucas"}},
	}
	_, _ = cli.InsertMany(context.Background(), docs)

	var err error
	// update already exist record
	filter1 := bson.M{
		"name": "Alice",
	}
	update1 := bson.M{
		operator.Set: bson.M{
			"name": "Alice1",
			"age":  18,
		},
	}
	err = cli.UpdateOne(context.Background(), filter1, update1)
	ast.NoError(err)

	// error when not exist
	filter2 := bson.M{
		"name": "Lily",
	}
	update2 := bson.M{
		operator.Set: bson.M{
			"name": "Lily",
			"age":  20,
		},
	}
	err = cli.UpdateOne(context.Background(), filter2, update2)
	ast.Equal(err, ErrNoSuchDocuments)

	// filter is nil or wrong BSON Document format
	update3 := bson.M{
		"name": "Geek",
		"age":  21,
	}
	err = cli.UpdateOne(context.Background(), nil, update3)
	ast.Error(err)

	err = cli.UpdateOne(context.Background(), 1, update3)
	ast.Error(err)

	// update is nil or wrong BSON Document format
	filter4 := bson.M{
		"name": "Geek",
	}
	err = cli.UpdateOne(context.Background(), filter4, nil)
	ast.Error(err)

	err = cli.UpdateOne(context.Background(), filter4, 1)
	ast.Error(err)
}

func TestCollection_UpdateAll(t *testing.T) {
	ast := require.New(t)
	cli := initClient("test")
	defer cli.Close(context.Background())
	defer cli.DropCollection(context.Background())
	cli.EnsureIndexes(context.Background(), nil, []string{"name"})

	id1 := primitive.NewObjectID()
	id2 := primitive.NewObjectID()
	id3 := primitive.NewObjectID()
	docs := []interface{}{
		bson.D{{Key: "_id", Value: id1}, {Key: "name", Value: "Alice"}, {Key: "age", Value: 18}},
		bson.D{{Key: "_id", Value: id2}, {Key: "name", Value: "Alice"}, {Key: "age", Value: 19}},
		bson.D{{Key: "_id", Value: id3}, {Key: "name", Value: "Lucas"}, {Key: "age", Value: 20}},
	}
	_, _ = cli.InsertMany(context.Background(), docs)

	var err error
	// update already exist record
	filter1 := bson.M{
		"name": "Alice",
	}
	update1 := bson.M{
		operator.Set: bson.M{
			"age": 33,
		},
	}
	res, err := cli.UpdateAll(context.Background(), filter1, update1)
	ast.NoError(err)
	ast.NotEmpty(res)
	ast.Equal(int64(2), res.MatchedCount)
	ast.Equal(int64(2), res.ModifiedCount)
	ast.Equal(int64(0), res.UpsertedCount)
	ast.Equal(nil, res.UpsertedID)

	// if record is not exist，err is nil， MatchedCount in res is 0
	filter2 := bson.M{
		"name": "Lily",
	}
	update2 := bson.M{
		operator.Set: bson.M{
			"age": 22,
		},
	}
	res, err = cli.UpdateAll(context.Background(), filter2, update2)
	ast.Nil(err)
	ast.NotNil(res)
	ast.Equal(int64(0), res.MatchedCount)

	// filter is nil or wrong BSON Document format
	update3 := bson.M{
		"name": "Geek",
		"age":  21,
	}
	res, err = cli.UpdateAll(context.Background(), nil, update3)
	ast.Error(err)
	ast.Nil(res)

	res, err = cli.UpdateAll(context.Background(), 1, update3)
	ast.Error(err)
	ast.Nil(res)

	// update is nil or wrong BSON Document format
	filter4 := bson.M{
		"name": "Geek",
	}
	res, err = cli.UpdateAll(context.Background(), filter4, nil)
	ast.Error(err)
	ast.Nil(res)

	res, err = cli.UpdateAll(context.Background(), filter4, 1)
	ast.Error(err)
	ast.Nil(res)
}

func TestCollection_Remove(t *testing.T) {
	ast := require.New(t)
	cli := initClient("test")
	defer cli.Close(context.Background())
	defer cli.DropCollection(context.Background())
	cli.EnsureIndexes(context.Background(), nil, []string{"name"})

	id1 := primitive.NewObjectID().Hex()
	id2 := primitive.NewObjectID().Hex()
	id3 := primitive.NewObjectID().Hex()
	id4 := primitive.NewObjectID().Hex()
	docs := []interface{}{
		bson.D{{Key: "_id", Value: id1}, {Key: "name", Value: "Alice"}, {Key: "age", Value: 18}},
		bson.D{{Key: "_id", Value: id2}, {Key: "name", Value: "Alice"}, {Key: "age", Value: 19}},
		bson.D{{Key: "_id", Value: id3}, {Key: "name", Value: "Lucas"}, {Key: "age", Value: 20}},
		bson.D{{Key: "_id", Value: id4}, {Key: "name", Value: "Joe"}, {Key: "age", Value: 20}},
	}
	_, _ = cli.InsertMany(context.Background(), docs)

	var err error
	// remove id
	err = cli.RemoveId(context.Background(), "")
	ast.Error(err)
	err = cli.RemoveId(context.Background(), "not-exists-id")
	ast.True(IsErrNoDocuments(err))
	ast.NoError(cli.RemoveId(context.Background(), id4))

	// delete record: name = "Alice" , after that, expect one name = "Alice" record
	filter1 := bson.M{
		"name": "Alice",
	}
	err = cli.Remove(context.Background(), filter1)
	ast.NoError(err)

	cnt, err := cli.Find(context.Background(), filter1).Count()
	ast.NoError(err)
	ast.Equal(int64(1), cnt)

	// delete not match  record , report err
	filter2 := bson.M{
		"name": "Lily",
	}
	err = cli.Remove(context.Background(), filter2)
	ast.Equal(err, ErrNoSuchDocuments)

	// filter is bson.M{}，delete one document
	filter3 := bson.M{}
	preCnt, err := cli.Find(context.Background(), filter3).Count()
	ast.NoError(err)
	ast.Equal(int64(2), preCnt)

	err = cli.Remove(context.Background(), filter3)
	ast.NoError(err)

	afterCnt, err := cli.Find(context.Background(), filter3).Count()
	ast.NoError(err)
	ast.Equal(preCnt-1, afterCnt)

	// filter is nil or wrong BSON Document format
	err = cli.Remove(context.Background(), nil)
	ast.Error(err)

	err = cli.Remove(context.Background(), 1)
	ast.Error(err)
}

func TestCollection_DeleteAll(t *testing.T) {
	ast := require.New(t)
	cli := initClient("test")
	defer cli.Close(context.Background())
	defer cli.DropCollection(context.Background())
	cli.EnsureIndexes(context.Background(), nil, []string{"name"})

	id1 := primitive.NewObjectID()
	id2 := primitive.NewObjectID()
	id3 := primitive.NewObjectID()
	id4 := primitive.NewObjectID()
	docs := []interface{}{
		bson.D{{Key: "_id", Value: id1}, {Key: "name", Value: "Alice"}, {Key: "age", Value: 18}},
		bson.D{{Key: "_id", Value: id2}, {Key: "name", Value: "Alice"}, {Key: "age", Value: 19}},
		bson.D{{Key: "_id", Value: id3}, {Key: "name", Value: "Lucas"}, {Key: "age", Value: 20}},
		bson.D{{Key: "_id", Value: id4}, {Key: "name", Value: "Rocket"}, {Key: "age", Value: 23}},
	}
	_, _ = cli.InsertMany(context.Background(), docs)

	var err error
	// delete record: name = "Alice" ,after that, expect - record : name = "Alice"
	filter1 := bson.M{
		"name": "Alice",
	}
	res, err := cli.RemoveAll(context.Background(), filter1)
	ast.NoError(err)
	ast.NotNil(res)
	ast.Equal(int64(2), res.DeletedCount)

	cnt, err := cli.Find(context.Background(), filter1).Count()
	ast.NoError(err)
	ast.Equal(int64(0), cnt)

	// delete with not match filter， DeletedCount in res is 0
	filter2 := bson.M{
		"name": "Lily",
	}
	res, err = cli.RemoveAll(context.Background(), filter2)
	ast.NoError(err)
	ast.NotNil(res)
	ast.Equal(int64(0), res.DeletedCount)

	// filter is bson.M{}，delete all docs
	filter3 := bson.M{}
	preCnt, err := cli.Find(context.Background(), filter3).Count()
	ast.NoError(err)
	ast.Equal(int64(2), preCnt)

	res, err = cli.RemoveAll(context.Background(), filter3)
	ast.NoError(err)
	ast.NotNil(res)
	ast.Equal(preCnt, res.DeletedCount)

	afterCnt, err := cli.Find(context.Background(), filter3).Count()
	ast.NoError(err)
	ast.Equal(int64(0), afterCnt)

	// filter is nil or wrong BSON Document format
	res, err = cli.RemoveAll(context.Background(), nil)
	ast.Error(err)
	ast.Nil(res)

	res, err = cli.RemoveAll(context.Background(), 1)
	ast.Error(err)
	ast.Nil(res)
}
func TestSliceInsert(t *testing.T) {
	newDocs := []UserInfo{{Name: "Alice", Age: 10}, {Name: "Lucas", Age: 11}}
	di := interface{}(newDocs)
	dis := interfaceToSliceInterface(di)
	ast := require.New(t)
	ast.Len(dis, 2)

	newDocs_1 := []interface{}{UserInfo{Name: "Alice", Age: 10}, UserInfo{Name: "Lucas", Age: 11}}
	di = interface{}(newDocs_1)
	dis = interfaceToSliceInterface(di)
	ast.Len(dis, 2)

	newDocs_2 := UserInfo{Name: "Alice", Age: 10}
	di = interface{}(newDocs_2)
	dis = interfaceToSliceInterface(di)
	ast.Nil(dis)

	newDocs_3 := []UserInfo{}
	di = interface{}(newDocs_3)
	dis = interfaceToSliceInterface(di)
	ast = require.New(t)
	ast.Nil(dis)
}
