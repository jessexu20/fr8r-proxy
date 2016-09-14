package auth

import (
	"net/http"
)

var validSpaces = []Creds{
///*swarmtest*/	{200, "10.140.28.132:2379", "", "", true, "85cdc7e0-32d8-4552-9bae-907c3f1d98d9", "swarm", "c3d87893a5b7f56991fd328f655f25cce286591c3ce4a558","9013217d-0abf-40fe-bd35-bb625066408c", "924fc412d1004528b90007e898aeb0d8", true},
///*swarm2test*/	{200, "10.140.179.44:2379", "", "", true, "c6549f25-1003-44c3-977d-09e866c4ea08", "swarm", "df6b2bd22dc7ed47f36f6e10a9f118c10f216ab44fb2863b","9013217d-0abf-40fe-bd35-bb625066408c", "8d174ad3d3a16169-BM_c6549f25-1003-44c3-977d-09e866c4ea08_8c5fd60fc2693a96", true},
///*swarm2test1*/	{200, "10.140.179.44:2379", "", "", true, "9994cfb1-cecb-4e17-9371-0c0b4fe5377b", "swarm", "01234567890","9013217d-0abf-40fe-bd35-bb625066408c", "01234567890",true},
//
///*swarm3test*/	{200, "10.140.181.167:2379", "", "", true, "eb651f47-3b4b-47d5-8880-1b47b70eaabd", "swarm", "01234567890","9013217d-0abf-40fe-bd35-bb625066408c", "01234567890", true},
///*swarm3test2*/	{200, "10.140.181.167:2379", "", "", true, "308a1b2d-b95b-4a07-a447-88af222ddd4c", "swarm", "01234567890","9013217d-0abf-40fe-bd35-bb625066408c", "01234567890", true},
///*devmon*/	{200, "10.140.181.167:2379", "", "", true, "416c1ae7-7cb1-40c5-84f6-1c0cdeff4303", "stage", "01234567890","a381cc61-0137-4042-8b55-2ccd3054882c", "01234567890", false},

}

// authentication stub for trusted SWARM tenants
// BlueMix space id is passed in request as X-Auth-Project-Id header
func StubAuth(r *http.Request) (creds Creds) {
	space_id := r.Header.Get("X-Auth-Project-Id")
	for _, v := range validSpaces {
		if space_id == v.Space_id {
			creds = v
			creds.Status = 200

			//Set Swarm Authorization header
			r.Header.Set("X-Auth-Token", space_id)

			creds.Docker_id = "" // --> no rewrite of docker_id  (rewrite is not needed with Swarm, needed in nova-docker case only)
			return
		}
	}
	creds.Status = 401
	return
}
