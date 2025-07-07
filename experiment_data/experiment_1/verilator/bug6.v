`timescale 1ns/1ps
module out63_block
(
    input  wire        clock_4 ,
    input  wire        clock_8 ,
    output wire [28:5] out63
);

reg [28:0]  reg_12;
reg [28:22] reg_24;

wire        _0558_;
wire [28:0] _0670_;
wire [28:0] _0399_;
wire        _0085_;
wire [28:0] _0769_;
wire        _0305_;
wire [23:0] _0306_;

assign _0558_ = | reg_24[26:25]; // reg_24 = 0 or 1100110 ---> _0558_ == 0
assign _0670_ = _0558_ ? reg_12 : 29'h00000f93; // _0558_ == 0 ---> _0670_ == 29'h00000f93
assign _0399_ = - _0670_; // _0670_ == 29'h00000f93 ---> _0399_ = 29'b11111111111111111000001101101
assign _0085_ = ~ _0399_[2]; // _0399_[2] == 1 ---> _0085_ == 0
assign { _0769_[28:3], _0769_[1:0] } = { _0399_[28:3], _0399_[1:0] }; // _0769_ != 0
assign _0769_[2] = _0085_; 
assign _0305_ = ! _0769_; // _0769_ != 0 ---> _0305_ == 0
assign _0306_ = ! _0305_; // _0305_ == 0 ---> _0306_ == 1
assign out63  = _0306_; // out63 == 1

always @(posedge clock_4, posedge clock_8)
    if (clock_8) reg_12 <= 29'h00000066;
    else reg_12 <= { reg_12[28:27], 25'h0000001, reg_12[1:0] };

always @(posedge clock_4, posedge clock_8)
    if (clock_8) reg_24 <= 7'h66;
    else reg_24 <= reg_24;

endmodule