package aws

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/autoscaling"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLaunchConfiguration() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLaunchConfigurationCreate,
		Read:   resourceAwsLaunchConfigurationRead,
		Delete: resourceAwsLaunchConfigurationDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"image_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"iam_instance_profile": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"key_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"user_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					switch v.(type) {
					case string:
						hash := sha1.Sum([]byte(v.(string)))
						return hex.EncodeToString(hash[:])
					default:
						return ""
					}
				},
			},

			"security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"associate_public_ip_address": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"spot_price": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"block_device": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Removed:  "Split out into three sub-types; see Changelog and Docs",
			},

			"ebs_block_device": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delete_on_termination": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
							ForceNew: true,
						},

						"device_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"iops": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"snapshot_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%t-", m["delete_on_termination"].(bool)))
					buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
					// NOTE: Not considering IOPS in hash; when using gp2, IOPS can come
					// back set to something like "33", which throws off the set
					// calculation and generates an unresolvable diff.
					// buf.WriteString(fmt.Sprintf("%d-", m["iops"].(int)))
					buf.WriteString(fmt.Sprintf("%s-", m["snapshot_id"].(string)))
					buf.WriteString(fmt.Sprintf("%d-", m["volume_size"].(int)))
					buf.WriteString(fmt.Sprintf("%s-", m["volume_type"].(string)))
					return hashcode.String(buf.String())
				},
			},

			"ephemeral_block_device": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"virtual_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["virtual_name"].(string)))
					return hashcode.String(buf.String())
				},
			},

			"root_block_device": &schema.Schema{
				// TODO: This is a set because we don't support singleton
				//       sub-resources today. We'll enforce that the set only ever has
				//       length zero or one below. When TF gains support for
				//       sub-resources this can be converted.
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					// "You can only modify the volume size, volume type, and Delete on
					// Termination flag on the block device mapping entry for the root
					// device volume." - bit.ly/ec2bdmap
					Schema: map[string]*schema.Schema{
						"delete_on_termination": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
							ForceNew: true,
						},

						"iops": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%t-", m["delete_on_termination"].(bool)))
					// See the NOTE in "ebs_block_device" for why we skip iops here.
					// buf.WriteString(fmt.Sprintf("%d-", m["iops"].(int)))
					buf.WriteString(fmt.Sprintf("%d-", m["volume_size"].(int)))
					buf.WriteString(fmt.Sprintf("%s-", m["volume_type"].(string)))
					return hashcode.String(buf.String())
				},
			},
		},
	}
}

func resourceAwsLaunchConfigurationCreate(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	// Figure out user data
	userData := ""
	if v := d.Get("user_data"); v != nil {
		userData = base64.StdEncoding.EncodeToString([]byte(v.(string)))
	}

	createLaunchConfigurationOpts := autoscaling.CreateLaunchConfigurationType{
		LaunchConfigurationName: aws.String(d.Get("name").(string)),
		ImageID:                 aws.String(d.Get("image_id").(string)),
		InstanceType:            aws.String(d.Get("instance_type").(string)),
		UserData:                aws.String(userData),
		EBSOptimized:            aws.Boolean(d.Get("ebs_optimized").(bool)),
		IAMInstanceProfile:      aws.String(d.Get("iam_instance_profile").(string)),
		PlacementTenancy:        aws.String(d.Get("placement_tenancy").(string)),
	}

	if v := d.Get("associate_public_ip_address"); v != nil {
		createLaunchConfigurationOpts.AssociatePublicIPAddress = aws.Boolean(v.(bool))
	} else {
		createLaunchConfigurationOpts.AssociatePublicIPAddress = aws.Boolean(false)
	}

	if v, ok := d.GetOk("key_name"); ok {
		createLaunchConfigurationOpts.KeyName = aws.String(v.(string))
	}
	if v, ok := d.GetOk("spot_price"); ok {
		createLaunchConfigurationOpts.SpotPrice = aws.String(v.(string))
	}

	if v, ok := d.GetOk("security_groups"); ok {
		createLaunchConfigurationOpts.SecurityGroups = expandStringList(
			v.(*schema.Set).List(),
		)
	}

	var blockDevices []autoscaling.BlockDeviceMapping

	if v, ok := d.GetOk("ebs_block_device"); ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			ebs := &autoscaling.EBS{
				DeleteOnTermination: aws.Boolean(bd["delete_on_termination"].(bool)),
			}

			if v, ok := bd["snapshot_id"].(string); ok && v != "" {
				ebs.SnapshotID = aws.String(v)
			}

			if v, ok := bd["volume_size"].(int); ok && v != 0 {
				ebs.VolumeSize = aws.Integer(v)
			}

			if v, ok := bd["volume_type"].(string); ok && v != "" {
				ebs.VolumeType = aws.String(v)
			}

			if v, ok := bd["iops"].(int); ok && v > 0 {
				ebs.IOPS = aws.Integer(v)
			}

			blockDevices = append(blockDevices, autoscaling.BlockDeviceMapping{
				DeviceName: aws.String(bd["device_name"].(string)),
				EBS:        ebs,
			})
		}
	}

	if v, ok := d.GetOk("ephemeral_block_device"); ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			blockDevices = append(blockDevices, autoscaling.BlockDeviceMapping{
				DeviceName:  aws.String(bd["device_name"].(string)),
				VirtualName: aws.String(bd["virtual_name"].(string)),
			})
		}
	}

	if v, ok := d.GetOk("root_block_device"); ok {
		vL := v.(*schema.Set).List()
		if len(vL) > 1 {
			return fmt.Errorf("Cannot specify more than one root_block_device.")
		}
		for _, v := range vL {
			bd := v.(map[string]interface{})
			ebs := &autoscaling.EBS{
				DeleteOnTermination: aws.Boolean(bd["delete_on_termination"].(bool)),
			}

			if v, ok := bd["volume_size"].(int); ok && v != 0 {
				ebs.VolumeSize = aws.Integer(v)
			}

			if v, ok := bd["volume_type"].(string); ok && v != "" {
				ebs.VolumeType = aws.String(v)
			}

			if v, ok := bd["iops"].(int); ok && v > 0 {
				ebs.IOPS = aws.Integer(v)
			}

			if dn, err := fetchRootDeviceName(d.Get("ami").(string), meta.(*AWSClient).ec2conn); err == nil {
				blockDevices = append(blockDevices, autoscaling.BlockDeviceMapping{
					DeviceName: dn,
					EBS:        ebs,
				})
			} else {
				return err
			}
		}
	}

	if len(blockDevices) > 0 {
		createLaunchConfigurationOpts.BlockDeviceMappings = blockDevices
	}

	log.Printf("[DEBUG] autoscaling create launch configuration: %#v", createLaunchConfigurationOpts)
	err := autoscalingconn.CreateLaunchConfiguration(&createLaunchConfigurationOpts)
	if err != nil {
		return fmt.Errorf("Error creating launch configuration: %s", err)
	}

	d.SetId(d.Get("name").(string))
	log.Printf("[INFO] launch configuration ID: %s", d.Id())

	// We put a Retry here since sometimes eventual consistency bites
	// us and we need to retry a few times to get the LC to load properly
	return resource.Retry(30*time.Second, func() error {
		return resourceAwsLaunchConfigurationRead(d, meta)
	})
}

func resourceAwsLaunchConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	describeOpts := autoscaling.LaunchConfigurationNamesType{
		LaunchConfigurationNames: []string{d.Id()},
	}

	log.Printf("[DEBUG] launch configuration describe configuration: %#v", describeOpts)
	describConfs, err := autoscalingconn.DescribeLaunchConfigurations(&describeOpts)
	if err != nil {
		return fmt.Errorf("Error retrieving launch configuration: %s", err)
	}
	if len(describConfs.LaunchConfigurations) == 0 {
		d.SetId("")
		return nil
	}

	// Verify AWS returned our launch configuration
	if *describConfs.LaunchConfigurations[0].LaunchConfigurationName != d.Id() {
		return fmt.Errorf(
			"Unable to find launch configuration: %#v",
			describConfs.LaunchConfigurations)
	}

	lc := describConfs.LaunchConfigurations[0]

	d.Set("key_name", *lc.KeyName)
	d.Set("image_id", *lc.ImageID)
	d.Set("instance_type", *lc.InstanceType)
	d.Set("name", *lc.LaunchConfigurationName)

	bds := make([]map[string]interface{}, len(lc.BlockDeviceMappings))
	for i, m := range lc.BlockDeviceMappings {
		bds[i] = make(map[string]interface{})
		bds[i]["device_name"] = m.DeviceName
		bds[i]["snapshot_id"] = m.EBS.SnapshotID
		bds[i]["volume_type"] = m.EBS.VolumeType
		bds[i]["volume_size"] = m.EBS.VolumeSize
		bds[i]["delete_on_termination"] = m.EBS.DeleteOnTermination
	}
	d.Set("block_device", bds)

	if lc.IAMInstanceProfile != nil {
		d.Set("iam_instance_profile", *lc.IAMInstanceProfile)
	} else {
		d.Set("iam_instance_profile", nil)
	}

	if lc.SpotPrice != nil {
		d.Set("spot_price", *lc.SpotPrice)
	} else {
		d.Set("spot_price", nil)
	}

	if lc.SecurityGroups != nil {
		d.Set("security_groups", lc.SecurityGroups)
	} else {
		d.Set("security_groups", nil)
	}

	return nil
}

func resourceAwsLaunchConfigurationDelete(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	log.Printf("[DEBUG] Launch Configuration destroy: %v", d.Id())
	err := autoscalingconn.DeleteLaunchConfiguration(
		&autoscaling.LaunchConfigurationNameType{LaunchConfigurationName: aws.String(d.Id())})
	if err != nil {
		autoscalingerr, ok := err.(aws.APIError)
		if ok && autoscalingerr.Code == "InvalidConfiguration.NotFound" {
			return nil
		}

		return err
	}

	return nil
}

func readBlockDevicesFromLaunchConfiguration(d *schema.ResourceData, launchConfiguration *autoscaling.LaunchConfiguration, autoscalingconn *autoscaling.AutoScaling) map[string]interface{} {
	var blockDevices = make(map[string]interface{})
	// Need to figure out how to determine the various instance types.
	return blockDevices
}
