package sponsor

// LICENSE

// Copyright (c) evcc.io (andig, naltatis, premultiply)

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"context"

	"github.com/evcc-io/evcc/api/proto/pb"
	"github.com/evcc-io/evcc/util/cloud"
	"github.com/evcc-io/evcc/util/request"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// checkHardware registers the device with the sponsor server and checks authorization.
func checkHardware(vendor string, metadata map[string]string) string {
	conn, err := cloud.Connection()
	if err != nil {
		return unavailable
	}

	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	client := pb.NewAuthClient(conn)
	res, err := client.IsAuthorizedHardware(ctx, &pb.HardwareRequest{
		MachineId: machineID(),
		Vendor:    vendor,
		Metadata:  metadata,
	})

	if err == nil && res.Authorized {
		return res.Subject
	}

	if s, ok := status.FromError(err); ok && s.Code() != codes.Unknown {
		return unavailable
	}

	return ""
}
