package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2transitgateway"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Only for testing purposes.
func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		vpc_a, err := ec2.NewVpc(ctx, "vpcA", &ec2.VpcArgs{
			CidrBlock:          pulumi.String("10.1.0.0/16"),
			EnableDnsHostnames: pulumi.BoolPtr(true),
			EnableDnsSupport:   pulumi.BoolPtr(true),
		})
		if err != nil {
			return err
		}

		igwa, err := ec2.NewInternetGateway(ctx, "igwa", &ec2.InternetGatewayArgs{
			VpcId: vpc_a.ID(),
		})
		if err != nil {
			return err
		}

		vpc_b, err := ec2.NewVpc(ctx, "vpcB", &ec2.VpcArgs{
			CidrBlock:          pulumi.String("10.2.0.0/16"),
			EnableDnsHostnames: pulumi.BoolPtr(true),
			EnableDnsSupport:   pulumi.BoolPtr(true),
		})
		if err != nil {
			return err
		}

		igwb, err := ec2.NewInternetGateway(ctx, "igwb", &ec2.InternetGatewayArgs{
			VpcId: vpc_b.ID(),
		})
		if err != nil {
			return err
		}

		tgw, err := ec2transitgateway.NewTransitGateway(ctx, "tgw", &ec2transitgateway.TransitGatewayArgs{})
		if err != nil {
			return err
		}

		rtablea, err := ec2.NewRouteTable(ctx, "rtableA", &ec2.RouteTableArgs{
			VpcId: vpc_a.ID(),
			Routes: ec2.RouteTableRouteArray{
				ec2.RouteTableRouteArgs{
					CidrBlock: pulumi.String("0.0.0.0/0"),
					GatewayId: igwa.ID(),
				},
				ec2.RouteTableRouteArgs{
					CidrBlock:        vpc_b.CidrBlock.ToStringOutput(),
					TransitGatewayId: tgw.ID(),
				},
			},
		})

		subnet_a, err := ec2.NewSubnet(ctx, "subnetA", &ec2.SubnetArgs{
			CidrBlock:        pulumi.String("10.1.1.0/24"),
			AvailabilityZone: pulumi.String("eu-west-1a"),
			VpcId:            vpc_a.ID(),
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, "rta-assoc", &ec2.RouteTableAssociationArgs{
			SubnetId:     subnet_a.ID(),
			RouteTableId: rtablea.ID(),
		})
		if err != nil {
			return err
		}

		rtableb, err := ec2.NewRouteTable(ctx, "rtableb", &ec2.RouteTableArgs{
			VpcId: vpc_b.ID(),
			Routes: ec2.RouteTableRouteArray{
				ec2.RouteTableRouteArgs{
					CidrBlock: pulumi.String("0.0.0.0/0"),
					GatewayId: igwb.ID(),
				},
				ec2.RouteTableRouteArgs{
					CidrBlock:        vpc_a.CidrBlock.ToStringOutput(),
					TransitGatewayId: tgw.ID(),
				},
			},
		})

		subnet_b, err := ec2.NewSubnet(ctx, "subnetB", &ec2.SubnetArgs{
			CidrBlock:        pulumi.String("10.2.1.0/24"),
			AvailabilityZone: pulumi.String("eu-west-1b"),
			VpcId:            vpc_b.ID(),
		})

		_, err = ec2.NewRouteTableAssociation(ctx, "rtb-assoc", &ec2.RouteTableAssociationArgs{
			SubnetId:     subnet_b.ID(),
			RouteTableId: rtableb.ID(),
		})
		if err != nil {
			return err
		}

		sga, err := ec2.NewSecurityGroup(ctx, "test-instance-a", &ec2.SecurityGroupArgs{
			VpcId: vpc_a.ID(),
			Ingress: ec2.SecurityGroupIngressArray{
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(22),
					ToPort:     pulumi.Int(22),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(80),
					ToPort:     pulumi.Int(80),
					CidrBlocks: pulumi.StringArray{subnet_b.CidrBlock.Elem()},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				ec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("-1"),
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
		})
		if err != nil {
			return err
		}

		sgb, err := ec2.NewSecurityGroup(ctx, "test-instance-b", &ec2.SecurityGroupArgs{
			VpcId: vpc_b.ID(),
			Ingress: ec2.SecurityGroupIngressArray{
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(22),
					ToPort:     pulumi.Int(22),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(80),
					ToPort:     pulumi.Int(80),
					CidrBlocks: pulumi.StringArray{subnet_a.CidrBlock.Elem()},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				ec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("-1"),
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
		})
		if err != nil {
			return err
		}

		userData := `#!/bin/bash
			sudo amazon-linux-extras install nginx1 -y
			sudo systemctl start nginx
			`
		eca, err := ec2.NewSpotInstanceRequest(ctx, "eca", &ec2.SpotInstanceRequestArgs{
			InstanceType:             pulumi.String("t4g.nano"),
			Ami:                      pulumi.String("ami-0f78f261d6135456a"),
			SubnetId:                 subnet_a.ID(),
			SecurityGroups:           pulumi.StringArray{sga.ID()},
			AssociatePublicIpAddress: pulumi.Bool(true),
			UserData:                 pulumi.String(userData),
			PrivateIp:                pulumi.String("10.1.1.5"),
		})
		if err != nil {
			return err
		}

		ecb, err := ec2.NewSpotInstanceRequest(ctx, "ecb", &ec2.SpotInstanceRequestArgs{
			InstanceType:             pulumi.String("t4g.nano"),
			Ami:                      pulumi.String("ami-0f78f261d6135456a"),
			SubnetId:                 subnet_b.ID(),
			SecurityGroups:           pulumi.StringArray{sgb.ID()},
			AssociatePublicIpAddress: pulumi.Bool(true),
			UserData:                 pulumi.String(userData),
			PrivateIp:                pulumi.String("10.2.1.5"),
		})
		if err != nil {
			return err
		}

		_, err = ec2transitgateway.NewVpcAttachment(ctx, "vpc-attachment-a", &ec2transitgateway.VpcAttachmentArgs{
			SubnetIds:        pulumi.StringArray{subnet_a.ID()},
			VpcId:            vpc_a.ID(),
			TransitGatewayId: tgw.ID(),
		})
		if err != nil {
			return err
		}

		_, err = ec2transitgateway.NewVpcAttachment(ctx, "vpc-attachment-b", &ec2transitgateway.VpcAttachmentArgs{
			SubnetIds:        pulumi.StringArray{subnet_b.ID()},
			VpcId:            vpc_b.ID(),
			TransitGatewayId: tgw.ID(),
		})
		if err != nil {
			return err
		}

		zone, err := route53.NewZone(ctx, "zone", &route53.ZoneArgs{
			Name: pulumi.String("marcus.local"),
			Vpcs: route53.ZoneVpcArray{
				route53.ZoneVpcArgs{
					VpcId:     vpc_a.ID(),
					VpcRegion: pulumi.String("eu-west-1"),
				},
				route53.ZoneVpcArgs{
					VpcId:     vpc_b.ID(),
					VpcRegion: pulumi.String("eu-west-1"),
				},
			},
		})
		if err != nil {
			return err
		}

		recorda, err := route53.NewRecord(ctx, "recorda", &route53.RecordArgs{
			Name:    pulumi.String("instancea"),
			ZoneId:  zone.ZoneId,
			Type:    pulumi.String("A"),
			Records: pulumi.StringArray{eca.PrivateIp},
			Ttl:     pulumi.Int(60),
		})
		if err != nil {
			return err
		}

		recordb, err := route53.NewRecord(ctx, "recordb", &route53.RecordArgs{
			Name:    pulumi.String("instanceb"),
			ZoneId:  zone.ZoneId,
			Type:    pulumi.String("A"),
			Records: pulumi.StringArray{ecb.PrivateIp},
			Ttl:     pulumi.Int(60),
		})
		if err != nil {
			return err
		}

		ctx.Export("eca", eca.ID())
		ctx.Export("ecb", ecb.ID())
		ctx.Export("tgw", tgw.ID())
		ctx.Export("zone", zone.ID())
		ctx.Export("recorda", recorda.ID())
		ctx.Export("recordb", recordb.ID())

		return nil
	})
}
