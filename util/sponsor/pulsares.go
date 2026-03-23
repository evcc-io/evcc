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
	"os"
	"strings"
	"time"
)

func readSerial() (string, error) {
	f, err := os.OpenFile("/dev/PulsaresSerial", os.O_RDWR, 0o644)
	if err != nil {
		return "", nil
	}

	if _, err := f.Write([]byte{0x0E, 0x00, 0x61, 0x7C, 0xD2, 0x71}); err != nil {
		return "", err
	}

	// serial timeout
	time.AfterFunc(3*time.Second, func() {
		_ = f.Close()
	})

	var token string
	b := make([]byte, 512)

	for {
		n, err := f.Read(b)
		if err != nil {
			return "", nil
		}

		token += string(b[:n])

		if token, ok := strings.CutSuffix(token, "\x04"); ok {
			return token, nil
		}
	}
}
