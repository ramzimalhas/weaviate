// Code generated by go-swagger; DO NOT EDIT.

package meta

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	models "github.com/creativesoftwarefdn/weaviate/models"
)

// WeaviateMetaGetReader is a Reader for the WeaviateMetaGet structure.
type WeaviateMetaGetReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *WeaviateMetaGetReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewWeaviateMetaGetOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	case 401:
		result := NewWeaviateMetaGetUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	default:
		return nil, runtime.NewAPIError("unknown error", response, response.Code())
	}
}

// NewWeaviateMetaGetOK creates a WeaviateMetaGetOK with default headers values
func NewWeaviateMetaGetOK() *WeaviateMetaGetOK {
	return &WeaviateMetaGetOK{}
}

/*WeaviateMetaGetOK handles this case with default header values.

Successful response.
*/
type WeaviateMetaGetOK struct {
	Payload *models.Meta
}

func (o *WeaviateMetaGetOK) Error() string {
	return fmt.Sprintf("[GET /meta][%d] weaviateMetaGetOK  %+v", 200, o.Payload)
}

func (o *WeaviateMetaGetOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Meta)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewWeaviateMetaGetUnauthorized creates a WeaviateMetaGetUnauthorized with default headers values
func NewWeaviateMetaGetUnauthorized() *WeaviateMetaGetUnauthorized {
	return &WeaviateMetaGetUnauthorized{}
}

/*WeaviateMetaGetUnauthorized handles this case with default header values.

Unauthorized or invalid credentials.
*/
type WeaviateMetaGetUnauthorized struct {
}

func (o *WeaviateMetaGetUnauthorized) Error() string {
	return fmt.Sprintf("[GET /meta][%d] weaviateMetaGetUnauthorized ", 401)
}

func (o *WeaviateMetaGetUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}