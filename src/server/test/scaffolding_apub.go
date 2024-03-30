package test

import (
	"fmt"
	"rss_parrot/dto"
	"strings"
	"sync"
	"time"
)

const birbPubKey = "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxOfKSoVvgObY5nih/3yd\nTlhF3w5etMIQZTvnsDLBQxRyvbEFfLoWOO2ug1tUq9XaIoagON+Fvrz75eFyHj87\nspeJ1dqgEGqtoAUiU0V1VZgn19iMdhVUWnTAtQobDMcs1Gs1tyHAOYSKTUadMXId\nYndLMQxHutXI9+ZWjLC322tbcYn9yuikJl58qQY8OtIG+Do4XJ5FuQKYMa11S3ff\ni+8I4Fp/3c6B4WviltC/FO41ntzgie/a9xNd9BkM9kattNvkMv0N3kkviG0KV1tq\nq+B1aiLFubaY27XTPvueJDzX39DeFl/+S/ak1rtkcoZXjMrqba4QvAFxFaFwjOrk\nhQIDAQAB\n-----END PUBLIC KEY-----"
const birbPrivKey = "-----BEGIN RSA PRIVATE KEY-----\nProc-Type: 4,ENCRYPTED\nDEK-Info: AES-256-CBC,FD72296B030A87484655DBF3F8ED44F4\n\nxmr4b9DTy3FJ5GuYMo3EU7wET6GJuUkWKYn9vDGWRGO1+UIGRWGBm7BQLogjahi7\ng57oXaQR9n1CTeD4azaBOSRNlNJ8lzlN3DMsr45wFVvtK8homAW84XuXlpzsC6N/\n8dqva0hF/S9OinCBIgKOJSwg4TEimNgpGpcCIeqXSfJiw+MEeF/r4g4EEViMKmmP\nyXdjFjbRhkAw9u/Av7rN3Dvoapd80Bp/8ZdZY5ofueO4G6ZPplPgJf0EkCJQAUMD\nhp5Z2lZ/w8B4U1yvuEvF+uNb2kxkJ7COhzruCsxZqt/gUdqUtWay8Vo0QaxeMdRf\n+dJmK+V5m+fFKXDL0h50A4D2IcQdgJ/MZYZUxXEPxldZPTUjriJ7HV/4b5rFolEl\nF750+2ngWCtvfskxq2ouszdM8QE4UkkUdbYwbPFFSbI8hX+TO9noAbVAyGXx79mz\n+gNhi8UQs3I+myms8mVEHGz8zVbj9CyNX/AcoEz54ngMuv9cPm4YASk0n2MOu/TH\nt2wx7H3swaG2jrGGursDjvo00i2qJbUvdnyTBsxBlxUPPk1IVZfHJyw8zUCxy1Je\nfeOBIFPKhzDK6aw8CIfrMawc8hU5bo6qb59Bf/QTGU3XfoklgG/EiGEv5k3n9q40\nmeVxNNG8nGjCiIR+D94vedKhSVyDk1G+4KeY0EG9HfAGbroj+HRghY3pjx0qEl9D\navQVsWDzaqyAS1Kan701dB7JfvYpHVoVG/Ica16xCA9Rfi/iyA6JFppNO/KJqf+u\nrpjRSPaHvXKFVSYAJlKZaAOOoQBsWHxvPifxe8cEWDgD0/+36TYCKGfh/LvA4fdc\nBvRri6FTbyOGePY/f3zcDA5AvTcgOv2f3loSvFg4euWzNC6886FrH1iRtRXn5uAG\nKKUz2F5YLFxsqnxGNEn5jAsjrZdPSt3B/RJR6KgM2eulJt6vgp0NeGiPh7kG+DVZ\nh5H1azr55zUrWfGB3pX19MHSudy1BYpW2iPPKX4SD1Up+Spfsq/3uBxXqIE9FRdR\nZ0RTiRH7uhEQCZfiN1qMrlQawrEEgyuNA0B+6uBqHbT4YsLQlkiE7XpOJFo1+gBS\n9NjQPwzVyGzPVQ5+sfFzn+VmiNoGOVrdRxrBXn0FzhioEdBCAR/mE2YecyrpeTZI\nEgZxNqrcsJz5/t9mehboc54QiHQclFMzltElY5BymXMn2WytWLT21lMjohCuXHa9\nwZGtL7SMeag1RtJ0jfYfTnxATkSmPViPQDKPr6hG0CHjWY+iGpcrkqzvD5RBGUC0\naX+XZwjHUgtZnLvMrwL5LIMWi8ttzgGU7ddRzEIIR0GnNPScNFLhqEsntDfyuYAJ\nwP78JmAxTk1W8xLc1suf10qygPdZXHHouFmlygq9ibqazO50zZiRsszN/99mN8Q7\nZss5q9G3bz4RevDeuYm8KghL0KmaeeuS/dj6kaqMvxh03B0CaMf6OHN9RYt3KcPC\nXQMdDoNd1+CKJxEvOm0VpulTCzYPNeEtdydYs5WHgvEuLVQnGFgJz22P/spwyvpd\nkf/Z6jUukjY0WyMCh1iypa8mF8hRzxaFXaJUbcaprjl17KkZ5RtWt7WQ2lgDgWTA\n-----END RSA PRIVATE KEY-----"
const accountPubKey1 = "-----BEGIN RSA PUBLIC KEY-----\nMIIBCgKCAQEAw0qPyECHmb5hpxMm6sdohub1LhbCOnKqO2KB39g0qo77YZmToaQR\njOyWPQuXj14dmUVO9sk3n1uMmFlo5sR2sCk4SSjwtbe9TDrfmDjQHYOGd90TdmGW\ngx8Pv5RtDI0ZcC4viXcrJN7V13bGt69nLK+xa/qyGyofxKwqHszsUaU1iyNML9cR\n3PmvlLBdsI/cUZNtt1bY8YRrUiUsogGFq/LFJ5zMP/lAMzTYif24oclby+aiPzJb\nbmcwPoroNuVaqTc8ePKiU/ZVgfnoUpAYfaRHWFK3RJ9A2rZQx66J7LZrDA29PowK\nejkxxchJBJ2lmQhEYlWUitrvtWT9DE+QcQIDAQAB\n-----END RSA PUBLIC KEY-----\n"
const accountPrivKey1 = "-----BEGIN RSA PRIVATE KEY-----\nProc-Type: 4,ENCRYPTED\nDEK-Info: AES-256-CBC,6c858c83d97a89d580310e3ad663ed40\n\n5ubfn7wV6WSvBOitypjZUpD24lGWypqV8YrrCsMDROHcwE9BMqRTv1xuhlaCI84g\nJiy2LP2yW7q8hNSBwOHJNCiWyfMOStOpAUVS1BdUCMZOnR5vFNLL1fpqugZThsHC\nlHx+ozwgEkya8PKACjmuyrMMyOdV9J7f3/L/PeAjYD7bVoHUFpGDvAYfEy6kmEcn\nXmjtjlk1PtokBVpN4306acmYkOMr6K46VyvtNPydnLY18vOjX0ok8Sa0W/7yAMsk\nucc8pIlpTEsT7jqZ5Vxlvb0LoYKnnQRfDFo2QfPLgTbDfxREvivq9lzjlY6ZvzK7\nBePdMZIxK+yghNKrGUsCWNysx8A1mRliIzSiohiqI4OFjq624IoSKEPh44bmjwCt\n7pZ2JyNkSRrKWenumgOZwbkOVfqqI3Z3VGjcB8PY/D52MkCvi+pRjSgqRq0e9QqE\nMKNTyBN59R5y86ofqSod8cf++M44oh/A8s9pqI5TSmNoaSzsiINZWbdfXjMmsT95\nGtRkIuOa2QPGTc5AD7Nzf5iLD39kTe9BKPf/NWEg5+jeCBf0Gm7CVOqwnbKMou0i\nkpfiTxQz+N+Ajc1M71DNEpvoiVmty9sHSox/nXdEhXm34qqTo8aIfQwwamcoj1em\n6ejCKKDt3nFvTcspWdUfFSjKxz38teaOEKiSqIAvfAzXarH5t985apjGPFM7lE5S\n33zLzNXP8J68OYHqPAqFV1HCD6z1V+tzNSZAXLTA8LbsDO6x92jpaeIzZWCnxHfQ\nHTlAwE68rznd5HAedi8Z85PoB7/tiqhPA8Ss4wtqCkG7xgcKCst/r1DBoR01uEUN\nb1Rg/546IdmptP/Vxsl/T3A9VnKO8rk0LmGqEchZuoPd2uScvr9vIS1j4rqveYBv\nGLF+oKKaPmzFc8SNt7Kp8oOTWWfTTLVglBpNLl7HAtv+VWlyqjPA6/aR7Mt1bUsE\n16JgcwHuj8BSyW6irSXzS1eU2EJj/aHuDRL30xLpPQQ1sp99lEIB8HFh9dr1v9aY\n3MMo012+Jnl0rRH2qPVUCQ1tUKvnFcfTaQEFg6AtYqhwMXnkcoWNXNPkITXW6qQO\nXEc7yRVt7hqGzbij0TZdlLFOJN2BL51H4P5QCbsXVbQZqXab25gfpMgGz5z03h2u\nxIsNgZWvNy/CCWW6eFIPtztee9zkid+/BR+/FQMkQ9kIFy4WbwH1D4yeWgStQQoV\nNK5SlzwAD/oLMCRfu1tT+NKZEa+GqBaIzoBtgTj9QqkGuf5O9/+DPPP49AoihZLK\nQtUq3dKV/GHmTSsMn+3rYaQLhWx76WnzgMHe8U3HHnOYzydGW5uc7KtOKrHgBpyE\nNQGS3IAjkBWei5e8DkaDj93+iV2Gn/vTFjU3/0OTWD5tVKC7SB+VpPRH8BeCZOA+\nZZCM2yccrDCNSjEu4y0brtnNtbc/1fqIGktzk6XJAjdlQlINtFy3jVIpnQWbR0DF\nb/j+cEYJvmW6ZLwMgYmZkyfxjVAfm/gximmWw6AuEmPVSzUZNz5o3BvyDQvqzDwW\nS00doj19t589uT+OxYAkWAxCCbmYGMdOFhRN2VB2ofpNbHb/cbdB/TUXKI1Ipf/u\n-----END RSA PRIVATE KEY-----\n"
const callerPubKey1 = "-----BEGIN RSA PUBLIC KEY-----\nMIIBCgKCAQEAvWcnbRP7l4XTHPhNIflGlfJNflcAeivqzAQdMwFu15EYYoXX9+ti\nGx75UpHDwT1wAJADCaAzzUrfeoB6RRuyPxrY437Iz7rd+jRet29rBf4+OQ/mxfLw\n6svgS56yPM+r/Rp6vwyGHjEI6u/Jz6Xa1LMBHB1XCnuzqstLj16UaxvBWWBPi+6A\nEPlP6HYIknrNjvoA3TPb64dufvXojQgoMbcnCf3h/SJ2JKvDLVAUsPvuXkvf9y+a\n8TcT6T6gJdsXszZ3rXI4JnmyAwOrmPvMcC3u5bS3jh+srEbrkXq0uOQ55lpwoDve\nkDIjk4OIjESuev6WFyv9IMhSWQrdBPxQwwIDAQAB\n-----END RSA PUBLIC KEY-----\n"
const callerPrivKey1 = "-----BEGIN RSA PRIVATE KEY-----\nProc-Type: 4,ENCRYPTED\nDEK-Info: AES-256-CBC,0bdae56bb96ff51da8e6a14a69489f22\n\ntedtuF5dsx5AAw712OVHayGkoMbOVIJBF4Khvm++6gtZhinGsFxT8hNWYWh374cZ\nYVAip/wRNDQZP2c/mLsi5t1esq0c/ul2D8Xc1vTUxqX5eVVYLFgnQ10EfkY8ct9P\nQtdmPG3VVbIrKWfaPq/TtZgsQ17ikwNZ6zI51N1GMIxt271IoQjnyF1MQ5sbnMuu\nfcLKp4OarGHicQJBcHIu6QXCNHdr5kc5uDIHKo1Mj2BR7p2K+RohlrJT1ZJtdVmB\nUCdoHM/kyuMaZgQz/h77+e72k1Recx6CMMIY7hDONAbwWPt0SVD36X+7gg+FapEu\nBx9HJJYd+xv72XsaettnLT95kAwEu21UzeTYDdweurI+WWTdZJXyVNKKPIwzx+ta\n681iA63il4bg4VoYNJSN09CTbLEf6MImXd2MOmQbQfe4ofxMrDCMmNVqvVA8DNMP\nFcvAA91SoMd4laaAk74ZZBLRhJG17BCo/vK1oee4922hhddar3r5o6fzk57d7d7K\nDM4VuVgHSt1241pDZiT0uCImkRilcjZ0IaPQL9ux/qQzKzDFWNYs2PgsF79RxbmR\nc3cH7Z5yqBz5fw8byepa6vuV9VWi7VryrcB/gO7buEunb5xHpEmX72bOPYCLwgel\ntC9cPHoT4GUmHbkaFxeiYFnx2lVOYC51XlwBpyJPRIn+i+BO0kVXtinuvkSYh4Ct\nRfkA0B3gyed69UrulLPBwoSYvNntXdoRkfMoxq/9SUvM9M6elj9TjhVzPtBZWCye\npgsKr744ODxcfjdGNv5gHxZ22k8/Oil4yOcxMDmXN4Oy9iimJRYb6wAkLaso3Btm\nMaw8zk3YgirI010QDaE5eU1HQ9Cn1Ol6wtOAM0YWVFVzv/YyGBop7yYrSTc6iOPe\nqS2bWWDPhUDjUhDANCOVkkF2KmDi9/P1Wd/3YCWgllHDfOmkj6bOEVXggEnzzrYd\nhaVFmjg3wSxh4XruoQRxXrRRSBKhkA4Ld+Mxf4kgVLtEKSYQD4qgJN2zlofwOtc8\n1wDCAe2tAVDEwqMv+MxHHsWpCAAM5ltXdyRcFb2cwMspoFUa7ICobfO2P8c0UGdr\niQ7Tnc6+buqXYurpDFsh8XPcuu00sGClMwp+El3Md9zLj4E7Gu8zEm7XCl1U5ycv\nZRaM4O5kHAOub7w+cX/uW1QLb8EDB1PGC9hJeT3hLio4T1n70hMVGZThO1z9adgO\nWeHTmZo8xoPTM1KwfP8w/Sp4OrEtpD1hXpy9ubiu9bC7tJEt7gTWYBXDN9NcHN3U\nXqKhC9jrTJULnU0v0zPk60hyYpLYg7UVi+od1QaBSMHPGdFawtoOouHqQcjLeWlR\nSpSfBdTcVcWlZWwbTIgtg5x1TuGwThBpiS3LZi2nG3X06XvOVlMQ/aSSs/GFpsCI\nRtE29pii+h2o7h9xrak8PNsyUX/waeQbpqUBtOD7KmC+IOCJ2IGFXAZJLVAeNu2g\n45uqlXH6+WSlLvpHiSTTFfWpOf1xA0djAJ4zgUgQYno630zbNbyeiIsjFjokiIo0\neGHF8xxCkEosLoAlNN+85gmUD0NmCWjTKltj3bWMgi8LKV8i1PSBvVf87YF/dltT\n-----END RSA PRIVATE KEY-----\n"
const callerPubKey2 = "-----BEGIN RSA PUBLIC KEY-----\nMIIBCgKCAQEA4Y9mAYAI05z+GGJkqp+aXfpTCp/joUGGalL1YCsIEZrvA1lcVe8G\nh6pdBgfeUxSRoBbyQpUs282sx18+PoHY89nHLLECGpdEMOltKEs7FRBbD0FfPVXT\nEh+pb3EjBuuA0wCgeeQ/1wOJAyi9bGyi0vFWltWOauCaYQCVnoM4K4HRfPSKN5Gj\nyyLihe97/xwWBmVJcEwD66xgPkPrGp9uwCszqXHm6w5of/YUhYLlnmQZ6cAtYgUH\nH5Qxvi4L1Zun3QW4fIfjHpH//wsnEyZtan5bXjiHClAjyIr3RLmwmgF2hScgisOc\noSwxopCsrPeoky3fyr0zGtxJEk3GzMmq9wIDAQAB\n-----END RSA PUBLIC KEY-----\n"
const callerPrivKey2 = "-----BEGIN RSA PRIVATE KEY-----\nProc-Type: 4,ENCRYPTED\nDEK-Info: AES-256-CBC,e454860384844cbc8f70d8018c5f9e6b\n\nS4KQT/0t59g1RIaCFisaQAQivK2sOh221vD4ZDt84TUEeyi/7fkeRbiS4z437uTU\n7+MOnEvp6qch+WCEpxjgNM6AQG1wsOA8UFs0duywpOt92qk9s3y6Qmj9qDTDqXNe\nO8ifFCsL1vzsHtE5xvbdIrOktyaDAO+reegWNwHyBxaMQGvqQRXpGvfEs0PR9HBL\nD+k2FR3H7DzI9s1R4ZNVpAwG865q1YgCKVlHzA4L9PIqR4XfPeb0b0Xk1q7ezqoF\nanB/IqRca4IApv+1BFwRNtb+NcG+Ak3qtmV8QINA3tJ25T/lcufmMwvbugVHbuk2\nWyxDTdAwNzobR0yhPMa3e+UfFgf3wYa4lNAaCzNTzl6YSoGFidhuz897ncDO0IE0\nnp3VTFMl9BNytgk9njRqVU8c4bUS9efB913KZHLcP6LbOuEjwLNbvHeLIvOUjgP5\n/CTSXSTYpf/NCgfjho/d2tdYM/gOC48ksrTZMArxt0+0ZaDZOze3Tl/zgUJZQh1b\nH27yWf+LfR3VG7nzusRcDdfVz05vsOn1nRz0pmCbFbAsnydGT5Q+mpnTarR9nQuL\n9y3ceH63SqZQhb+RG3hQ/MlAW8LWz99oiONQGoWLny5Ww13qlIepttYvKDw/OQHQ\ngRd2tu6MPsTSGiObJe73BfH9K143WQ1LRrHL2odNEQrqaGyV87QU0V7uqa+A0Acv\nr03zSQGI+44HSJ16uknZhKIJFKCHRqKxuhd7MUd4/Cz0P4f2W5yVfFqvXYUy/B+l\n6kDtZa+8ZHCzJIpwvjVYTaSbdNUN+WMHQGW/b7fMDAZeleSRYFiWSUR1Umm0pwp2\nm8snlwaoX0ua8MWdPHy49kp+EZW9Kjjc9vC6myx9mPNsTaBsO9rvAUTc9aTILwfr\nHS866wrIh16vSrAyvHJfb0/RIym+sA6W6RBRCiNE0qlyMJW19rtT/D8nIGteKFwU\nmcORKg/Roe6UOT4HTWVcea7vgbHaOTzpbZh+I0pFF0l6ijsJ7tMFOXs9uki0KI63\n6J9bZZVvM3jqIwhgT0nW5qWhwLqLQTtebe4GyvR0ztzV9pFV/Rkn4om1x4Jx3gr/\nVdTn+gKGa7Ugv3QB4GU+42Z+LWvW5vRpT1wiOrP2oM3UfG4d6RrIXrNHEtlB1eXX\nmbh4nfRQ2mX8caWErZdko9oJ+Ufe95nogObnG5EACuWW+/726hBlGJzvuHvuSwrh\nLGPSv/YlcdPNgM+1FMXnE47EdwVT4hf0+vUqv2Fe5NHktaO7IO5nXbLn1rCGcZCV\n8ZhBSTIhPFZ7jAJw3WpoxsMO2h8pjCoTWzLxPRSukBvQIdywdtH8hgellevvdBZ4\nCLqGsvNnCRFOcxyk6xegjEWSWwnErCKQ3VTlFmPotNzBRIpJ0G0521tdLhHqipwM\nf8dbob4Ggkb0lkU7mUHgBejwDS5Hgw3M1gXiUsnoUPgZ5aVEVlGq9WvTrN3rFVUg\nWoyIe5r7eXz3I92hRHlMSjs9swejetQl7a2eJH4sqrzhxevxeq8kbYdRNBP9WcEB\n3eDlFP72i/XxRE0losaR3Suv5rqFaFmXvmJqTc2gWezr2evl+PgcBPTUko8AClEO\n-----END RSA PRIVATE KEY-----\n"

var muId sync.Mutex
var id int64 = time.Now().UnixNano()

func getNextId() uint64 {
	var res int64
	muId.Lock()
	res = id
	id += 1
	muId.Unlock()
	return uint64(res)
}

func makeCallerUserInfo(host, name, pubKeyPem string) *dto.UserInfo {
	return &dto.UserInfo{
		Context:           struct{}{},
		Id:                fmt.Sprintf("https://%s/users/%s", host, name),
		Type:              "Person",
		PreferredUserName: name,
		Name:              name,
		Summary:           "Account bio",
		ManuallyApproves:  false,
		Published:         "2023-12-10T00:00:00Z",
		Inbox:             fmt.Sprintf("https://%s/users/%s/inbox", host, name),
		Outbox:            fmt.Sprintf("https://%s/users/%s/outbox", host, name),
		Followers:         fmt.Sprintf("https://%s/users/%s/followers", host, name),
		Following:         fmt.Sprintf("https://%s/users/%s/following", host, name),
		Endpoints: dto.UserEndpoints{
			SharedInbox: fmt.Sprintf("https://%s/inbox", host),
		},
		PublicKey: dto.PublicKey{
			Id:           fmt.Sprintf("https://%s/users/%s#main-key", host, name),
			Owner:        fmt.Sprintf("https://%s/users/%s", host, name),
			PublicKeyPem: pubKeyPem,
		},
		Attachments: []dto.Attachment{},
		Icon:        dto.Image{},
		Image:       dto.Image{},
	}
}

func makeCreateNote(host, name, content string, to, cc []string, inReplyTo *string, tags string) []byte {
	bytes, err := fs.ReadFile("data/create-note.json")
	if err != nil {
		panic(err)
	}
	json := string(bytes)

	actor := fmt.Sprintf("https://%s/users/%s", host, name)
	largeId := fmt.Sprintf("%d", getNextId())

	json = strings.ReplaceAll(json, "{{activity-id}}", fmt.Sprintf("%s/statuses/%s", actor, largeId))
	json = strings.ReplaceAll(json, "{{actor}}", actor)
	json = strings.ReplaceAll(json, "{{published}}", time.Now().UTC().Format(time.RFC3339))
	json = strings.ReplaceAll(json, "{{content}}", content)
	json = strings.ReplaceAll(json, "{{tags}}", tags)
	if inReplyTo == nil {
		json = strings.ReplaceAll(json, "{{inReplyTo}}", "null")
	} else {
		json = strings.ReplaceAll(json, "{{inReplyTo}}", "\""+*inReplyTo+"\"")
	}

	listToStr := func(list []string) string {
		if len(list) == 1 {
			return `"` + list[0] + `"`
		}
		res := "["
		for ix, s := range list {
			if ix > 0 {
				res += ", "
			}
			res += `"` + s + `"`
		}
		res += "]"
		return res
	}

	json = strings.ReplaceAll(json, "{{to}}", listToStr(to))
	json = strings.ReplaceAll(json, "{{cc}}", listToStr(cc))
	return []byte(json)
}
